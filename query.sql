-- name: GetMission :one
SELECT m.id, m.name, m.drone_id, m.image_folder, d.ip as drone_ip, s.id as seq_id FROM missions m
JOIN drones d ON m.drone_id = d.id
JOIN sequences s ON m.seq_id = s.id
WHERE m.id = $1 LIMIT 1;

-- name: ListMissions :many
SELECT 
	m.id,
	m.name,
	m.package_id,
	m.status,
	p.name AS package_name,
	m.drone_id,
	d.name AS drone_name,
	m.seq_id,
	s.name AS seq_name
FROM 
missions m
JOIN drones d ON m.drone_id = d.id
JOIN packages p ON m.package_id = p.id
JOIN sequences s ON m.seq_id = s.id;

-- name: InsertMission :one
INSERT INTO missions (
		name,
		drone_id,
		package_id,
		seq_id,
		image_folder,
		status,
		path
) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7
) RETURNING *;

-- name: DeleteMission :one
DELETE FROM missions
WHERE id = $1
RETURNING *;

-- name: UpdateMissionStatus :one
UPDATE missions
SET status = $1
WHERE id = $2
RETURNING *;

-- name: UpdateMissionImageFolder :one
UPDATE missions
SET image_folder = $1
WHERE id = $2
RETURNING *;
-- name: InsertPackage :one
INSERT INTO packages (
		name,
		weight,
		height,
		length,
		sender_id,
		receiver_id
) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6
) RETURNING *;

-- name: ListPackages :many
SELECT
    p.id,
    p.name,
    p.weight,
    p.height,
    p.length,
    s.name AS sender_name,
    r.name AS receiver_name
FROM
    packages p
JOIN
    users s ON p.sender_id = s.id
JOIN
    users r ON p.receiver_id = r.id;

-- name: InsertDrone :one
INSERT INTO drones (
		name,
		address,
		ip,
		status,
		port
) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
) RETURNING *;

-- name: ListDrones :many
SELECT * FROM drones;

-- name: ListActiveDrones :many
SELECT * FROM drones
WHERE status = true;

-- name: GetDroneByID :one
SELECT * FROM drones
WHERE id = $1 LIMIT 1;

-- name: ResetAllDroneStatus :many
UPDATE drones
SET status = false
RETURNING *;

-- name: InsertSequence :one
INSERT INTO sequences (
		name,
		description,
		seq,
		length
) VALUES (
		$1,
		$2,
		$3,
		$4
) RETURNING *;

-- name: ListSequences :many
SELECT * FROM sequences;

-- name: GetSequenceByID :one
SELECT * FROM sequences
WHERE id = $1 LIMIT 1;

-- name: DeleteDrone :many
DELETE FROM drones
WHERE id = $1
RETURNING name;

-- name: InsertUser :one
INSERT INTO users (
		id,
		name
) VALUES (
		$1,
		$2
) RETURNING *;

-- name: ListUsers :many
SELECT * FROM users;

-- name: DeleteSequence :many
DELETE FROM sequences
WHERE id = $1
RETURNING *;
