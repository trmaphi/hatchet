// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0
// source: queue.sql

package dbsqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const bulkQueueItems = `-- name: BulkQueueItems :exec
UPDATE
    "QueueItem" qi
SET
    "isQueued" = false
WHERE
    qi."id" = ANY($1::bigint[])
`

func (q *Queries) BulkQueueItems(ctx context.Context, db DBTX, ids []int64) error {
	_, err := db.Exec(ctx, bulkQueueItems, ids)
	return err
}

const cleanupInternalQueueItems = `-- name: CleanupInternalQueueItems :exec
DELETE FROM "InternalQueueItem"
WHERE "isQueued" = 'f'
AND
    "id" >= $1::bigint
    AND "id" <= $2::bigint
    AND "tenantId" = $3::uuid
`

type CleanupInternalQueueItemsParams struct {
	Minid    int64       `json:"minid"`
	Maxid    int64       `json:"maxid"`
	Tenantid pgtype.UUID `json:"tenantid"`
}

func (q *Queries) CleanupInternalQueueItems(ctx context.Context, db DBTX, arg CleanupInternalQueueItemsParams) error {
	_, err := db.Exec(ctx, cleanupInternalQueueItems, arg.Minid, arg.Maxid, arg.Tenantid)
	return err
}

const cleanupQueueItems = `-- name: CleanupQueueItems :exec
DELETE FROM "QueueItem"
WHERE "isQueued" = 'f'
AND
    "id" >= $1::bigint
    AND "id" <= $2::bigint
    AND "tenantId" = $3::uuid
`

type CleanupQueueItemsParams struct {
	Minid    int64       `json:"minid"`
	Maxid    int64       `json:"maxid"`
	Tenantid pgtype.UUID `json:"tenantid"`
}

func (q *Queries) CleanupQueueItems(ctx context.Context, db DBTX, arg CleanupQueueItemsParams) error {
	_, err := db.Exec(ctx, cleanupQueueItems, arg.Minid, arg.Maxid, arg.Tenantid)
	return err
}

const cleanupRetryQueueItems = `-- name: CleanupRetryQueueItems :exec
DELETE FROM "RetryQueueItem"
WHERE "isQueued" = 'f'
AND
    "retryAfter" >= $1::timestamp
    AND "retryAfter" <= $2::timestamp
    AND "tenantId" = $3::uuid
`

type CleanupRetryQueueItemsParams struct {
	Minretryafter pgtype.Timestamp `json:"minretryafter"`
	Maxretryafter pgtype.Timestamp `json:"maxretryafter"`
	Tenantid      pgtype.UUID      `json:"tenantid"`
}

func (q *Queries) CleanupRetryQueueItems(ctx context.Context, db DBTX, arg CleanupRetryQueueItemsParams) error {
	_, err := db.Exec(ctx, cleanupRetryQueueItems, arg.Minretryafter, arg.Maxretryafter, arg.Tenantid)
	return err
}

const cleanupTimeoutQueueItems = `-- name: CleanupTimeoutQueueItems :exec
DELETE FROM "TimeoutQueueItem"
WHERE "isQueued" = 'f'
AND
    "id" >= $1::bigint
    AND "id" <= $2::bigint
    AND "tenantId" = $3::uuid
`

type CleanupTimeoutQueueItemsParams struct {
	Minid    int64       `json:"minid"`
	Maxid    int64       `json:"maxid"`
	Tenantid pgtype.UUID `json:"tenantid"`
}

func (q *Queries) CleanupTimeoutQueueItems(ctx context.Context, db DBTX, arg CleanupTimeoutQueueItemsParams) error {
	_, err := db.Exec(ctx, cleanupTimeoutQueueItems, arg.Minid, arg.Maxid, arg.Tenantid)
	return err
}

const createInternalQueueItemsBulk = `-- name: CreateInternalQueueItemsBulk :exec
INSERT INTO
    "InternalQueueItem" (
        "queue",
        "isQueued",
        "data",
        "tenantId",
        "priority"
    )
SELECT
    input."queue",
    true,
    input."data",
    input."tenantId",
    1
FROM (
    SELECT
        unnest(cast($1::text[] as"InternalQueue"[])) AS "queue",
        unnest($2::json[]) AS "data",
        unnest($3::uuid[]) AS "tenantId"
) AS input
ON CONFLICT DO NOTHING
`

type CreateInternalQueueItemsBulkParams struct {
	Queues    []string      `json:"queues"`
	Datas     [][]byte      `json:"datas"`
	Tenantids []pgtype.UUID `json:"tenantids"`
}

func (q *Queries) CreateInternalQueueItemsBulk(ctx context.Context, db DBTX, arg CreateInternalQueueItemsBulkParams) error {
	_, err := db.Exec(ctx, createInternalQueueItemsBulk, arg.Queues, arg.Datas, arg.Tenantids)
	return err
}

const createQueueItem = `-- name: CreateQueueItem :exec
INSERT INTO
    "QueueItem" (
        "stepRunId",
        "stepId",
        "actionId",
        "scheduleTimeoutAt",
        "stepTimeout",
        "priority",
        "isQueued",
        "tenantId",
        "queue",
        "sticky",
        "desiredWorkerId"
    )
VALUES
    (
        $1::uuid,
        $2::uuid,
        $3::text,
        $4::timestamp,
        $5::text,
        COALESCE($6::integer, 1),
        true,
        $7::uuid,
        $8,
        $9::"StickyStrategy",
        $10::uuid
    )
`

type CreateQueueItemParams struct {
	StepRunId         pgtype.UUID        `json:"stepRunId"`
	StepId            pgtype.UUID        `json:"stepId"`
	ActionId          pgtype.Text        `json:"actionId"`
	ScheduleTimeoutAt pgtype.Timestamp   `json:"scheduleTimeoutAt"`
	StepTimeout       pgtype.Text        `json:"stepTimeout"`
	Priority          pgtype.Int4        `json:"priority"`
	Tenantid          pgtype.UUID        `json:"tenantid"`
	Queue             string             `json:"queue"`
	Sticky            NullStickyStrategy `json:"sticky"`
	DesiredWorkerId   pgtype.UUID        `json:"desiredWorkerId"`
}

func (q *Queries) CreateQueueItem(ctx context.Context, db DBTX, arg CreateQueueItemParams) error {
	_, err := db.Exec(ctx, createQueueItem,
		arg.StepRunId,
		arg.StepId,
		arg.ActionId,
		arg.ScheduleTimeoutAt,
		arg.StepTimeout,
		arg.Priority,
		arg.Tenantid,
		arg.Queue,
		arg.Sticky,
		arg.DesiredWorkerId,
	)
	return err
}

type CreateQueueItemsBulkParams struct {
	StepRunId         pgtype.UUID        `json:"stepRunId"`
	StepId            pgtype.UUID        `json:"stepId"`
	ActionId          pgtype.Text        `json:"actionId"`
	ScheduleTimeoutAt pgtype.Timestamp   `json:"scheduleTimeoutAt"`
	StepTimeout       pgtype.Text        `json:"stepTimeout"`
	Priority          int32              `json:"priority"`
	IsQueued          bool               `json:"isQueued"`
	TenantId          pgtype.UUID        `json:"tenantId"`
	Queue             string             `json:"queue"`
	Sticky            NullStickyStrategy `json:"sticky"`
	DesiredWorkerId   pgtype.UUID        `json:"desiredWorkerId"`
}

const createRetryQueueItem = `-- name: CreateRetryQueueItem :exec
INSERT INTO
    "RetryQueueItem" (
        "stepRunId",
        "retryAfter",
        "tenantId",
        "isQueued"
    )
VALUES
    (
        $1::uuid,
        $2::timestamp,
        $3::uuid,
        true
    )
`

type CreateRetryQueueItemParams struct {
	Steprunid  pgtype.UUID      `json:"steprunid"`
	Retryafter pgtype.Timestamp `json:"retryafter"`
	Tenantid   pgtype.UUID      `json:"tenantid"`
}

func (q *Queries) CreateRetryQueueItem(ctx context.Context, db DBTX, arg CreateRetryQueueItemParams) error {
	_, err := db.Exec(ctx, createRetryQueueItem, arg.Steprunid, arg.Retryafter, arg.Tenantid)
	return err
}

const createTimeoutQueueItem = `-- name: CreateTimeoutQueueItem :exec
INSERT INTO
    "InternalQueueItem" (
        "stepRunId",
        "retryCount",
        "timeoutAt",
        "tenantId",
        "isQueued"
    )
SELECT
    $1::uuid,
    $2::integer,
    $3::timestamp,
    $4::uuid,
    true
ON CONFLICT DO NOTHING
`

type CreateTimeoutQueueItemParams struct {
	Steprunid  pgtype.UUID      `json:"steprunid"`
	Retrycount int32            `json:"retrycount"`
	Timeoutat  pgtype.Timestamp `json:"timeoutat"`
	Tenantid   pgtype.UUID      `json:"tenantid"`
}

func (q *Queries) CreateTimeoutQueueItem(ctx context.Context, db DBTX, arg CreateTimeoutQueueItemParams) error {
	_, err := db.Exec(ctx, createTimeoutQueueItem,
		arg.Steprunid,
		arg.Retrycount,
		arg.Timeoutat,
		arg.Tenantid,
	)
	return err
}

const createUniqueInternalQueueItemsBulk = `-- name: CreateUniqueInternalQueueItemsBulk :exec
INSERT INTO
    "InternalQueueItem" (
        "queue",
        "isQueued",
        "data",
        "tenantId",
        "priority",
        "uniqueKey"
    )
SELECT
    $1::"InternalQueue",
    true,
    input."data",
    $2::uuid,
    1,
    input."uniqueKey"
FROM (
    SELECT
        unnest($3::json[]) AS "data",
        unnest($4::text[]) AS "uniqueKey"
) AS input
ON CONFLICT DO NOTHING
`

type CreateUniqueInternalQueueItemsBulkParams struct {
	Queue      InternalQueue `json:"queue"`
	Tenantid   pgtype.UUID   `json:"tenantid"`
	Datas      [][]byte      `json:"datas"`
	Uniquekeys []string      `json:"uniquekeys"`
}

func (q *Queries) CreateUniqueInternalQueueItemsBulk(ctx context.Context, db DBTX, arg CreateUniqueInternalQueueItemsBulkParams) error {
	_, err := db.Exec(ctx, createUniqueInternalQueueItemsBulk,
		arg.Queue,
		arg.Tenantid,
		arg.Datas,
		arg.Uniquekeys,
	)
	return err
}

const getMinMaxProcessedInternalQueueItems = `-- name: GetMinMaxProcessedInternalQueueItems :one
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId",
    COALESCE(MAX("id"), 0)::bigint AS "maxId"
FROM
    "InternalQueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = $1::uuid
`

type GetMinMaxProcessedInternalQueueItemsRow struct {
	MinId int64 `json:"minId"`
	MaxId int64 `json:"maxId"`
}

func (q *Queries) GetMinMaxProcessedInternalQueueItems(ctx context.Context, db DBTX, tenantid pgtype.UUID) (*GetMinMaxProcessedInternalQueueItemsRow, error) {
	row := db.QueryRow(ctx, getMinMaxProcessedInternalQueueItems, tenantid)
	var i GetMinMaxProcessedInternalQueueItemsRow
	err := row.Scan(&i.MinId, &i.MaxId)
	return &i, err
}

const getMinMaxProcessedQueueItems = `-- name: GetMinMaxProcessedQueueItems :one
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId",
    COALESCE(MAX("id"), 0)::bigint AS "maxId"
FROM
    "QueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = $1::uuid
`

type GetMinMaxProcessedQueueItemsRow struct {
	MinId int64 `json:"minId"`
	MaxId int64 `json:"maxId"`
}

func (q *Queries) GetMinMaxProcessedQueueItems(ctx context.Context, db DBTX, tenantid pgtype.UUID) (*GetMinMaxProcessedQueueItemsRow, error) {
	row := db.QueryRow(ctx, getMinMaxProcessedQueueItems, tenantid)
	var i GetMinMaxProcessedQueueItemsRow
	err := row.Scan(&i.MinId, &i.MaxId)
	return &i, err
}

const getMinMaxProcessedRetryQueueItems = `-- name: GetMinMaxProcessedRetryQueueItems :one
SELECT
    COALESCE(MIN("retryAfter"), NOW())::timestamp AS "minRetryAfter",
    COALESCE(MAX("retryAfter"), NOW())::timestamp AS "maxRetryAfter"
FROM
    "RetryQueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = $1::uuid
`

type GetMinMaxProcessedRetryQueueItemsRow struct {
	MinRetryAfter pgtype.Timestamp `json:"minRetryAfter"`
	MaxRetryAfter pgtype.Timestamp `json:"maxRetryAfter"`
}

func (q *Queries) GetMinMaxProcessedRetryQueueItems(ctx context.Context, db DBTX, tenantid pgtype.UUID) (*GetMinMaxProcessedRetryQueueItemsRow, error) {
	row := db.QueryRow(ctx, getMinMaxProcessedRetryQueueItems, tenantid)
	var i GetMinMaxProcessedRetryQueueItemsRow
	err := row.Scan(&i.MinRetryAfter, &i.MaxRetryAfter)
	return &i, err
}

const getMinMaxProcessedTimeoutQueueItems = `-- name: GetMinMaxProcessedTimeoutQueueItems :one
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId",
    COALESCE(MAX("id"), 0)::bigint AS "maxId"
FROM
    "TimeoutQueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = $1::uuid
`

type GetMinMaxProcessedTimeoutQueueItemsRow struct {
	MinId int64 `json:"minId"`
	MaxId int64 `json:"maxId"`
}

func (q *Queries) GetMinMaxProcessedTimeoutQueueItems(ctx context.Context, db DBTX, tenantid pgtype.UUID) (*GetMinMaxProcessedTimeoutQueueItemsRow, error) {
	row := db.QueryRow(ctx, getMinMaxProcessedTimeoutQueueItems, tenantid)
	var i GetMinMaxProcessedTimeoutQueueItemsRow
	err := row.Scan(&i.MinId, &i.MaxId)
	return &i, err
}

const getMinUnprocessedQueueItemId = `-- name: GetMinUnprocessedQueueItemId :one
WITH priority_1 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = $1::uuid
        AND "queue" = $2::text
        AND "priority" = 1
    ORDER BY
        "id" ASC
    LIMIT 1
),
priority_2 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = $1::uuid
        AND "queue" = $2::text
        AND "priority" = 2
    ORDER BY
        "id" ASC
    LIMIT 1
),
priority_3 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = $1::uuid
        AND "queue" = $2::text
        AND "priority" = 3
    ORDER BY
        "id" ASC
    LIMIT 1
),
priority_4 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = $1::uuid
        AND "queue" = $2::text
        AND "priority" = 4
    ORDER BY
        "id" ASC
    LIMIT 1
)
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId"
FROM (
    SELECT "id" FROM priority_1
    UNION ALL
    SELECT "id" FROM priority_2
    UNION ALL
    SELECT "id" FROM priority_3
    UNION ALL
    SELECT "id" FROM priority_4
) AS combined_priorities
`

type GetMinUnprocessedQueueItemIdParams struct {
	Tenantid pgtype.UUID `json:"tenantid"`
	Queue    string      `json:"queue"`
}

func (q *Queries) GetMinUnprocessedQueueItemId(ctx context.Context, db DBTX, arg GetMinUnprocessedQueueItemIdParams) (int64, error) {
	row := db.QueryRow(ctx, getMinUnprocessedQueueItemId, arg.Tenantid, arg.Queue)
	var minId int64
	err := row.Scan(&minId)
	return minId, err
}

const getQueuedCounts = `-- name: GetQueuedCounts :many
SELECT
    "queue",
    COUNT(*) AS "count"
FROM
    "QueueItem" qi
WHERE
    qi."isQueued" = true
    AND qi."tenantId" = $1::uuid
GROUP BY
    qi."queue"
`

type GetQueuedCountsRow struct {
	Queue string `json:"queue"`
	Count int64  `json:"count"`
}

func (q *Queries) GetQueuedCounts(ctx context.Context, db DBTX, tenantid pgtype.UUID) ([]*GetQueuedCountsRow, error) {
	rows, err := db.Query(ctx, getQueuedCounts, tenantid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*GetQueuedCountsRow
	for rows.Next() {
		var i GetQueuedCountsRow
		if err := rows.Scan(&i.Queue, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listActionsForAvailableWorkers = `-- name: ListActionsForAvailableWorkers :many
SELECT
    w."id" as "workerId",
    a."actionId"
FROM
    "Worker" w
JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
JOIN
    "Action" a ON atw."A" = a."id"
WHERE
    w."tenantId" = $1::uuid
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false
`

type ListActionsForAvailableWorkersRow struct {
	WorkerId pgtype.UUID `json:"workerId"`
	ActionId string      `json:"actionId"`
}

func (q *Queries) ListActionsForAvailableWorkers(ctx context.Context, db DBTX, tenantid pgtype.UUID) ([]*ListActionsForAvailableWorkersRow, error) {
	rows, err := db.Query(ctx, listActionsForAvailableWorkers, tenantid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListActionsForAvailableWorkersRow
	for rows.Next() {
		var i ListActionsForAvailableWorkersRow
		if err := rows.Scan(&i.WorkerId, &i.ActionId); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listActionsForWorkers = `-- name: ListActionsForWorkers :many
SELECT
    w."id" as "workerId",
    a."actionId"
FROM
    "Worker" w
LEFT JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
LEFT JOIN
    "Action" a ON atw."A" = a."id"
WHERE
    w."tenantId" = $1::uuid
    AND w."id" = ANY($2::uuid[])
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false
`

type ListActionsForWorkersParams struct {
	Tenantid  pgtype.UUID   `json:"tenantid"`
	Workerids []pgtype.UUID `json:"workerids"`
}

type ListActionsForWorkersRow struct {
	WorkerId pgtype.UUID `json:"workerId"`
	ActionId pgtype.Text `json:"actionId"`
}

func (q *Queries) ListActionsForWorkers(ctx context.Context, db DBTX, arg ListActionsForWorkersParams) ([]*ListActionsForWorkersRow, error) {
	rows, err := db.Query(ctx, listActionsForWorkers, arg.Tenantid, arg.Workerids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListActionsForWorkersRow
	for rows.Next() {
		var i ListActionsForWorkersRow
		if err := rows.Scan(&i.WorkerId, &i.ActionId); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listActiveWorkers = `-- name: ListActiveWorkers :many
SELECT
    w."id",
    w."maxRuns"
FROM
    "Worker" w
WHERE
    w."tenantId" = $1::uuid
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false
`

type ListActiveWorkersRow struct {
	ID      pgtype.UUID `json:"id"`
	MaxRuns int32       `json:"maxRuns"`
}

func (q *Queries) ListActiveWorkers(ctx context.Context, db DBTX, tenantid pgtype.UUID) ([]*ListActiveWorkersRow, error) {
	rows, err := db.Query(ctx, listActiveWorkers, tenantid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListActiveWorkersRow
	for rows.Next() {
		var i ListActiveWorkersRow
		if err := rows.Scan(&i.ID, &i.MaxRuns); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listAllAvailableSlotsForWorkers = `-- name: ListAllAvailableSlotsForWorkers :many
WITH worker_max_runs AS (
    SELECT
        "id",
        "maxRuns"
    FROM
        "Worker"
    WHERE
        "tenantId" = $1::uuid
), worker_filled_slots AS (
    SELECT
        "workerId",
        COUNT("stepRunId") AS "filledSlots"
    FROM
        "SemaphoreQueueItem"
    WHERE
        "tenantId" = $1::uuid
    GROUP BY
        "workerId"
)
SELECT
    wmr."id",
    wmr."maxRuns" - COALESCE(wfs."filledSlots", 0) AS "availableSlots"
FROM
    worker_max_runs wmr
LEFT JOIN
    worker_filled_slots wfs ON wmr."id" = wfs."workerId"
`

type ListAllAvailableSlotsForWorkersRow struct {
	ID             pgtype.UUID `json:"id"`
	AvailableSlots int32       `json:"availableSlots"`
}

// subtract the filled slots from the max runs to get the available slots
func (q *Queries) ListAllAvailableSlotsForWorkers(ctx context.Context, db DBTX, tenantid pgtype.UUID) ([]*ListAllAvailableSlotsForWorkersRow, error) {
	rows, err := db.Query(ctx, listAllAvailableSlotsForWorkers, tenantid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListAllAvailableSlotsForWorkersRow
	for rows.Next() {
		var i ListAllAvailableSlotsForWorkersRow
		if err := rows.Scan(&i.ID, &i.AvailableSlots); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listAllWorkerActions = `-- name: ListAllWorkerActions :many
SELECT
    a."actionId" AS actionId
FROM "Worker" w
LEFT JOIN "_ActionToWorker" aw ON w.id = aw."B"
LEFT JOIN "Action" a ON aw."A" = a.id
WHERE
    a."tenantId" = $1::uuid AND
    w."id" = $2::uuid
`

type ListAllWorkerActionsParams struct {
	Tenantid pgtype.UUID `json:"tenantid"`
	Workerid pgtype.UUID `json:"workerid"`
}

func (q *Queries) ListAllWorkerActions(ctx context.Context, db DBTX, arg ListAllWorkerActionsParams) ([]pgtype.Text, error) {
	rows, err := db.Query(ctx, listAllWorkerActions, arg.Tenantid, arg.Workerid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []pgtype.Text
	for rows.Next() {
		var actionid pgtype.Text
		if err := rows.Scan(&actionid); err != nil {
			return nil, err
		}
		items = append(items, actionid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listAvailableSlotsForWorkers = `-- name: ListAvailableSlotsForWorkers :many
WITH worker_max_runs AS (
    SELECT
        "id",
        "maxRuns"
    FROM
        "Worker"
    WHERE
        "tenantId" = $1::uuid
        AND "id" = ANY($2::uuid[])
), worker_filled_slots AS (
    SELECT
        "workerId",
        COUNT("stepRunId") AS "filledSlots"
    FROM
        "SemaphoreQueueItem"
    WHERE
        "tenantId" = $1::uuid
        AND "workerId" = ANY($2::uuid[])
    GROUP BY
        "workerId"
)
SELECT
    wmr."id",
    wmr."maxRuns" - COALESCE(wfs."filledSlots", 0) AS "availableSlots"
FROM
    worker_max_runs wmr
LEFT JOIN
    worker_filled_slots wfs ON wmr."id" = wfs."workerId"
`

type ListAvailableSlotsForWorkersParams struct {
	Tenantid  pgtype.UUID   `json:"tenantid"`
	Workerids []pgtype.UUID `json:"workerids"`
}

type ListAvailableSlotsForWorkersRow struct {
	ID             pgtype.UUID `json:"id"`
	AvailableSlots int32       `json:"availableSlots"`
}

// subtract the filled slots from the max runs to get the available slots
func (q *Queries) ListAvailableSlotsForWorkers(ctx context.Context, db DBTX, arg ListAvailableSlotsForWorkersParams) ([]*ListAvailableSlotsForWorkersRow, error) {
	rows, err := db.Query(ctx, listAvailableSlotsForWorkers, arg.Tenantid, arg.Workerids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListAvailableSlotsForWorkersRow
	for rows.Next() {
		var i ListAvailableSlotsForWorkersRow
		if err := rows.Scan(&i.ID, &i.AvailableSlots); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listInternalQueueItems = `-- name: ListInternalQueueItems :many
SELECT
    id, queue, "isQueued", data, "tenantId", priority, "uniqueKey"
FROM
    "InternalQueueItem" qi
WHERE
    qi."isQueued" = true
    AND qi."tenantId" = $1::uuid
    AND qi."queue" = $2::"InternalQueue"
    AND (
        $3::bigint IS NULL OR
        qi."id" >= $3::bigint
    )
    -- Added to ensure that the index is used
    AND qi."priority" >= 1 AND qi."priority" <= 4
ORDER BY
    qi."priority" DESC,
    qi."id" ASC
LIMIT
    COALESCE($4::integer, 100)
FOR UPDATE SKIP LOCKED
`

type ListInternalQueueItemsParams struct {
	Tenantid pgtype.UUID   `json:"tenantid"`
	Queue    InternalQueue `json:"queue"`
	GtId     pgtype.Int8   `json:"gtId"`
	Limit    pgtype.Int4   `json:"limit"`
}

func (q *Queries) ListInternalQueueItems(ctx context.Context, db DBTX, arg ListInternalQueueItemsParams) ([]*InternalQueueItem, error) {
	rows, err := db.Query(ctx, listInternalQueueItems,
		arg.Tenantid,
		arg.Queue,
		arg.GtId,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*InternalQueueItem
	for rows.Next() {
		var i InternalQueueItem
		if err := rows.Scan(
			&i.ID,
			&i.Queue,
			&i.IsQueued,
			&i.Data,
			&i.TenantId,
			&i.Priority,
			&i.UniqueKey,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listQueueItemsForQueue = `-- name: ListQueueItemsForQueue :many
SELECT
    qi.id, qi."stepRunId", qi."stepId", qi."actionId", qi."scheduleTimeoutAt", qi."stepTimeout", qi.priority, qi."isQueued", qi."tenantId", qi.queue, qi.sticky, qi."desiredWorkerId",
    sr."status"
FROM
    "QueueItem" qi
JOIN
    "StepRun" sr ON qi."stepRunId" = sr."id"
WHERE
    qi."isQueued" = true
    AND qi."tenantId" = $1::uuid
    AND qi."queue" = $2::text
    AND (
        $3::bigint IS NULL OR
        qi."id" >= $3::bigint
    )
    -- Added to ensure that the index is used
    AND qi."priority" >= 1 AND qi."priority" <= 4
ORDER BY
    qi."priority" DESC,
    qi."id" ASC
LIMIT
    COALESCE($4::integer, 100)
`

type ListQueueItemsForQueueParams struct {
	Tenantid pgtype.UUID `json:"tenantid"`
	Queue    string      `json:"queue"`
	GtId     pgtype.Int8 `json:"gtId"`
	Limit    pgtype.Int4 `json:"limit"`
}

type ListQueueItemsForQueueRow struct {
	QueueItem QueueItem     `json:"queue_item"`
	Status    StepRunStatus `json:"status"`
}

func (q *Queries) ListQueueItemsForQueue(ctx context.Context, db DBTX, arg ListQueueItemsForQueueParams) ([]*ListQueueItemsForQueueRow, error) {
	rows, err := db.Query(ctx, listQueueItemsForQueue,
		arg.Tenantid,
		arg.Queue,
		arg.GtId,
		arg.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListQueueItemsForQueueRow
	for rows.Next() {
		var i ListQueueItemsForQueueRow
		if err := rows.Scan(
			&i.QueueItem.ID,
			&i.QueueItem.StepRunId,
			&i.QueueItem.StepId,
			&i.QueueItem.ActionId,
			&i.QueueItem.ScheduleTimeoutAt,
			&i.QueueItem.StepTimeout,
			&i.QueueItem.Priority,
			&i.QueueItem.IsQueued,
			&i.QueueItem.TenantId,
			&i.QueueItem.Queue,
			&i.QueueItem.Sticky,
			&i.QueueItem.DesiredWorkerId,
			&i.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listQueues = `-- name: ListQueues :many
SELECT
    id, "tenantId", name, "lastActive"
FROM
    "Queue"
WHERE
    "tenantId" = $1::uuid
    AND "lastActive" > NOW() - INTERVAL '1 day'
`

func (q *Queries) ListQueues(ctx context.Context, db DBTX, tenantid pgtype.UUID) ([]*Queue, error) {
	rows, err := db.Query(ctx, listQueues, tenantid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*Queue
	for rows.Next() {
		var i Queue
		if err := rows.Scan(
			&i.ID,
			&i.TenantId,
			&i.Name,
			&i.LastActive,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listStepRunsToRetry = `-- name: ListStepRunsToRetry :many
WITH retries AS (
    SELECT
        id, "retryAfter", "stepRunId", "tenantId", "isQueued"
    FROM
        "RetryQueueItem" rqi
    WHERE
        rqi."isQueued" = true
        AND rqi."tenantId" = $1::uuid
        AND rqi."retryAfter" <= NOW()
    ORDER BY
        rqi."retryAfter" ASC
    LIMIT
        1000
    FOR UPDATE SKIP LOCKED
), updated_rqis AS (
    UPDATE
        "RetryQueueItem" rqi
    SET
        "isQueued" = false
    FROM
        retries
    WHERE
        rqi."stepRunId" = retries."stepRunId"
)
SELECT
    retries.id, retries."retryAfter", retries."stepRunId", retries."tenantId", retries."isQueued"
FROM
    retries
JOIN
    "StepRun" sr ON retries."stepRunId" = sr."id"
WHERE
    -- we remove any step runs in a finalized state from the retry queue
    sr."status" NOT IN ('SUCCEEDED', 'FAILED', 'CANCELLED', 'CANCELLING')
`

type ListStepRunsToRetryRow struct {
	ID         int64            `json:"id"`
	RetryAfter pgtype.Timestamp `json:"retryAfter"`
	StepRunId  pgtype.UUID      `json:"stepRunId"`
	TenantId   pgtype.UUID      `json:"tenantId"`
	IsQueued   bool             `json:"isQueued"`
}

func (q *Queries) ListStepRunsToRetry(ctx context.Context, db DBTX, tenantid pgtype.UUID) ([]*ListStepRunsToRetryRow, error) {
	rows, err := db.Query(ctx, listStepRunsToRetry, tenantid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListStepRunsToRetryRow
	for rows.Next() {
		var i ListStepRunsToRetryRow
		if err := rows.Scan(
			&i.ID,
			&i.RetryAfter,
			&i.StepRunId,
			&i.TenantId,
			&i.IsQueued,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const markInternalQueueItemsProcessed = `-- name: MarkInternalQueueItemsProcessed :exec
UPDATE
    "InternalQueueItem" qi
SET
    "isQueued" = false
WHERE
    qi."id" = ANY($1::bigint[])
`

func (q *Queries) MarkInternalQueueItemsProcessed(ctx context.Context, db DBTX, ids []int64) error {
	_, err := db.Exec(ctx, markInternalQueueItemsProcessed, ids)
	return err
}

const popTimeoutQueueItems = `-- name: PopTimeoutQueueItems :many
WITH qis AS (
    SELECT
        "id",
        "stepRunId"
    FROM
        "TimeoutQueueItem"
    WHERE
        "isQueued" = true
        AND "tenantId" = $1::uuid
        AND "timeoutAt" <= NOW()
    ORDER BY
        "timeoutAt" ASC
    LIMIT
        COALESCE($2::integer, 100)
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "TimeoutQueueItem" qi
SET
    "isQueued" = false
FROM
    qis
WHERE
    qi."id" = qis."id"
RETURNING
    qis."stepRunId" AS "stepRunId"
`

type PopTimeoutQueueItemsParams struct {
	Tenantid pgtype.UUID `json:"tenantid"`
	Limit    pgtype.Int4 `json:"limit"`
}

func (q *Queries) PopTimeoutQueueItems(ctx context.Context, db DBTX, arg PopTimeoutQueueItemsParams) ([]pgtype.UUID, error) {
	rows, err := db.Query(ctx, popTimeoutQueueItems, arg.Tenantid, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []pgtype.UUID
	for rows.Next() {
		var stepRunId pgtype.UUID
		if err := rows.Scan(&stepRunId); err != nil {
			return nil, err
		}
		items = append(items, stepRunId)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removeTimeoutQueueItem = `-- name: RemoveTimeoutQueueItem :exec
DELETE FROM
    "TimeoutQueueItem"
WHERE
    "stepRunId" = $1::uuid
    AND "retryCount" = $2::integer
`

type RemoveTimeoutQueueItemParams struct {
	Steprunid  pgtype.UUID `json:"steprunid"`
	Retrycount int32       `json:"retrycount"`
}

func (q *Queries) RemoveTimeoutQueueItem(ctx context.Context, db DBTX, arg RemoveTimeoutQueueItemParams) error {
	_, err := db.Exec(ctx, removeTimeoutQueueItem, arg.Steprunid, arg.Retrycount)
	return err
}

const upsertQueue = `-- name: UpsertQueue :exec
WITH queue_exists AS (
    SELECT
        1
    FROM
        "Queue"
    WHERE
        "tenantId" = $1::uuid
        AND "name" = $2::text
), queue_to_update AS (
    SELECT
        id, "tenantId", name, "lastActive"
    FROM
        "Queue"
    WHERE
        EXISTS (
            SELECT
                1
            FROM
                queue_exists
        )
        AND "tenantId" = $1::uuid
        AND "name" = $2::text
    FOR UPDATE SKIP LOCKED
), update_queue AS (
    UPDATE
        "Queue"
    SET
        "lastActive" = NOW()
    FROM
        queue_to_update
    WHERE
        "Queue"."tenantId" = queue_to_update."tenantId"
        AND "Queue"."name" = queue_to_update."name"
)
INSERT INTO
    "Queue" (
        "tenantId",
        "name",
        "lastActive"
    )
SELECT
    $1::uuid,
    $2::text,
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM queue_exists
)
ON CONFLICT ("tenantId", "name") DO NOTHING
`

type UpsertQueueParams struct {
	Tenantid pgtype.UUID `json:"tenantid"`
	Name     string      `json:"name"`
}

func (q *Queries) UpsertQueue(ctx context.Context, db DBTX, arg UpsertQueueParams) error {
	_, err := db.Exec(ctx, upsertQueue, arg.Tenantid, arg.Name)
	return err
}
