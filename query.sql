-- name: GetMission :one
SELECT * FROM missions
WHERE id = $1 LIMIT 1;

-- name: ListMission :many
SELECT * FROM missions;

-- name: CreateMission :one
INSERT INTO missions (
	protobuf
) VALUES ( 
	$1
)
RETURNING *;