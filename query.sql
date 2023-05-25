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
		ip,
		name
) VALUES (
		$1,
		$2,
		$3
) RETURNING *;

-- name: ListDrone: one
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