import asyncio
import asyncpg
from mavsdk import System
import websockets
import json
import requests
import aioredis


# Connect to Redis database
async def connect_to_redis():
		print("Connecting to the Redis database...")
		redis = await aioredis.from_url("redis://localhost:6379")
		print("Connected to the Redis database")
		return redis


# Connect to PostgreSQL database
async def connect_to_database():
		print("Connecting to the database...")
		conn = await asyncpg.connect(
				user="saltmurai",
				password="saltmurai",
				database="saltmurai",
				host="localhost",
				port="5432",
		)
		print("Connected to the database")
		return conn


# Query the drone system address from the database
async def get_drone_system_addresses(conn):
		print("Querying drone system addresses from the database...")
		query = "SELECT id, address FROM drones WHERE status = false"
		addresses = [record["address"] for record in await conn.fetch(query)]
		ids = [record["id"] for record in await conn.fetch(query)]
		print(ids)
		print("Drone system addresses retrieved from the database:", addresses)
		return addresses, ids


# Set the status of the drone to true in the database
async def update_drone_status(conn, address, port):
		print("Updating the drone status in the database...")
		query = "UPDATE drones SET status = true, port = $2  WHERE address = $1"
		await conn.execute(query, address, port)
		print("Drone status updated in the database")


# Set the status of the drone to false in the database
async def reset_drone_status(conn, address):
		print("Resetting the drone status in the database...")
		query = "UPDATE drones SET status = false WHERE address = $1"
		await conn.execute(query, address)
		print("Drone status reset in the database")


async def connect_to_drone(address, port):
		print("Connecting to the drone " + address)
		# create a new instance of the drone
		drone = System(port=port)
		try:
				await asyncio.wait_for(drone.connect(system_address=address), timeout=3)
				print("Connected to the drone")
				return drone
		except TimeoutError:
				raise TimeoutError("Connection to the drone timed out")
		except Exception as e:
				raise Exception("Connection to the drone failed")


async def get_position(drone, websocket, drone_id, r):
		data = {
			"drone_id": drone_id,
		}
		async for position in drone.telemetry.position():
				data["latitude_deg"] = position.latitude_deg
				data["longitude_deg"] = position.longitude_deg
				data["absolute_altitude_m"] = position.absolute_altitude_m
				data["relative_altitude_m"] = position.relative_altitude_m
				try:
						await websocket.send(json.dumps(data))
						await r.set(f"{str(drone_id)}-postion", json.dumps(data))
				except websockets.exceptions.ConnectionClosed:
						break
				except Exception as e:
						print(e)
				await asyncio.sleep(1)


async def get_battery(drone, websocket, id, r):
		data = {
			"drone_id": id,
		}
		async for battery in drone.telemetry.battery():
				data["voltage_v"] = battery.voltage_v
				data["remaining_percent"] = battery.remaining_percent
				try:
						# await websocket.send(json.dumps(data))
						await r.set(f"{str(id)}-battery", json.dumps(data))
				except websockets.exceptions.ConnectionClosed:
						break
				except Exception as e:
						print(e)
				await asyncio.sleep(1)


async def get_flight_mode(drone, websocket, id, r):
		data = {
			"drone_id": id,
		}
		async for flight_mode in drone.telemetry.flight_mode():
				data["flight_mode"] = f"${flight_mode}"
				try:
						# await websocket.send(json.dumps(data))
						await r.set(f"{str(id)}-flight_mode", json.dumps(data))
				except websockets.exceptions.ConnectionClosed:
						break
				except Exception as e:
						print(e)
				await asyncio.sleep(1)


# Forward telemetry data to all connected WebSocket clients
async def forward_telemetry_data(websocket, path):
		conn = await connect_to_database()
		r = await connect_to_redis()
		addresses, ids = await get_drone_system_addresses(conn)
		drones = []
		tasks = []
		start_port = 50051
		i = 0
		try:
				i = 0
				for address in addresses:
						try:
								drone = await connect_to_drone(address, start_port + i)
						except Exception as e:
								continue
						drones.append(drone)
						await update_drone_status(conn, address, port=start_port + i)
						i += 1
						# await update_drone_status(conn, address)

						position_task = asyncio.create_task(
								get_position(drone, websocket, ids[addresses.index(address)], r)
						)
						battery_task = asyncio.create_task(
								get_battery(drone, websocket, ids[addresses.index(address)], r)
						)
						flight_mode_task = asyncio.create_task(
								get_flight_mode(drone, websocket, ids[addresses.index(address)], r)
						)
						tasks.append(position_task)
						tasks.append(battery_task)

						tasks.append(flight_mode_task)

				await asyncio.gather(*tasks)
		except asyncio.CancelledError:
				print("Forwarding telemetry data cancelled")
		except Exception as e:
				print(e)
		finally:
				print("Cleaning up...")
				for drone in drones:
						drone._stop_mavsdk_server()
				for address in addresses:
						await reset_drone_status(conn, address)
				for id in ids:
						await r.delete(f"${str(id)}-postion")
						await r.delete(f"${str(id)}-battery")
						await r.delete(f"${str(id)}-flight_mode")


async def receive_client_messages(websocket, path):
		try:
				async for message in websocket:
						try:
								data = json.loads(message)
								print(data)
								action = data.get("action")
								system_address = data.get("system_address")
								port = data.get("port")
								if not action or not system_address or not port:
										print("Invalid message format.")
										continue
								
								drone = System(port=port)
								await drone.connect(system_address=system_address)
								# Perform actions based on the received message
								# You can add different if-else conditions here based on different actions
								# For example:
								if action == "return_home":
										try:
											await drone.action.return_to_launch()
										except Exception as e:
											print(e)
										# Perform some action on the drone with the given system_address
								elif action == "land":
										print(data)
										try:
											await drone.action.land()
										except Exception as e:
											print(e)
										# Perform some other action on the drone
								elif action == "takeoff":
										print(data)
										# Perform some other action on the drone
										try: 
											await drone.action.arm()
											await drone.action.takeoff()
										except Exception as e:
											print(e)
								elif action == "arm":
										try:
											await drone.action.arm()
										except Exception as e:
											print(e)
								elif action == "disarm":
										pass
								else:
										print("Unknown action received.")
										print(data)

						except json.JSONDecodeError as e:
								print("Error decoding JSON:", e)
						except Exception as e:
								print("Error processing message:", e)

		except websockets.ConnectionClosed:
				print("Client connection closed.")


# Main program
async def main(websocket, path):
		print("New WebSocket client connected")
		await asyncio.gather(
				forward_telemetry_data(websocket, path),
				receive_client_messages(websocket, path),
		)


# Set up WebSocket server
start_server = websockets.serve(main, "0.0.0.0", 3003)

print("WebSocket server started")
try:
		asyncio.get_event_loop().run_until_complete(start_server)
		asyncio.get_event_loop().run_forever()
except KeyboardInterrupt:
		print("WebSocket server stopped")
		print("Reseting drone status in the database...")
		# called POST api http://localhost:3002/resetDrones
		res = requests.post("http://localhost:3002/resetDrones")
		if res.status_code == 200:
				print("Drone status reset in the database")
