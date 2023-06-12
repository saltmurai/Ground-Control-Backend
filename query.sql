-- name: GetMission :one
SELECT * FROM missions
WHERE id = $1 LIMIT 1;

-- name: ListMission :many
SELECT * FROM missions;

-- name: InsertPackage :one
INSERT INTO packages (
		name,
		weight
) VALUES (
		$1,
		$2
) RETURNING *;

-- name: ListPackage :many
SELECT * FROM packages;

-- name: InsertDrone :one
INSERT INTO drones (
		id,
		name,
		address,
		status
) VALUES (
		$1,
		$2,
		$3,
		$4
) RETURNING *;

-- name: ListDrones :many
SELECT * FROM drones;

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

