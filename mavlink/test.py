import asyncio
from mavsdk import System


async def print_flight_mode():
    drone = System()
    await drone.connect(system_address="serial:///dev/ttyUSB0:57600")

    print("Waiting for drone to connect...")
    async for flight_mode in drone.telemetry.raw_gps():
        print("FlightMode:", flight_mode)


if __name__ == "__main__":
    # Start the main function
    asyncio.run(print_flight_mode())