-- name: GetMission :one
SELECT * FROM missions
WHERE id = $1 LIMIT 1;

-- name: ListMission :many
SELECT * FROM missions;

-- name: InsertMission :one
INSERT INTO missions (
		name,
		drone_id,
		package_id,
		seq_id,
		image_folder
) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
) RETURNING *;

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
		status
) VALUES (
		$1,
		$2,
		$3,
		$4
) RETURNING *;

-- name: ListDrones :many
SELECT * FROM drones;

-- name: ListActiveDrones :many
SELECT * FROM drones
WHERE status = true;

-- name: ResetAllDroneStatus :many
UPDATE drones
SET status = false
RETURNING *;

-- name: InsertSequence :one
INSERT INTO sequences (
		name,
		description,
		seq,
		created_at
) VALUES (
		$1,
		$2,
		$3,
		$4
) RETURNING *;

--name: ListSequences :many
SELECT * FROM sequences;

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

