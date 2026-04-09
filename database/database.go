package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DatabaseService struct {
	pool *pgxpool.Pool
}

func NewDatabaseService(databaseUrl string) *DatabaseService {
	config, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create pool: %v\n", err)
		os.Exit(1)
	}

	return &DatabaseService{
		pool: pool,
	}
}

func (h *DatabaseService) AddExpire(fileId int64, groupId int64) (time.Time, error) {
	var createdAt time.Time
	strFileId := strconv.FormatInt(fileId, 10)
	strGroupId := strconv.FormatInt(groupId, 10)

	query := `
		INSERT INTO "ExpireFile" ("fileId", "groupId") 
		VALUES ($1, $2) 
		RETURNING "createdAt"`

	err := h.pool.QueryRow(context.Background(), query, strFileId, strGroupId).Scan(&createdAt)
	if err != nil {
		return time.Time{}, err
	}

	return createdAt, nil
}

type ExpireRecord struct {
	FileID    int64
	GroupID   int64
	CreatedAt time.Time
}

func (h *DatabaseService) GetExpiredFiles() ([]ExpireRecord, error) {
	query := `
		SELECT "fileId", "groupId", "createdAt" 
		FROM "ExpireFile" 
		WHERE "createdAt" <= NOW() - INTERVAL '24 hours'
		LIMIT 100`

	rows, err := h.pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ExpireRecord
	for rows.Next() {
		var strFileId, strGroupId string
		var createdAt time.Time

		if err := rows.Scan(&strFileId, &strGroupId, &createdAt); err != nil {
			return nil, err
		}

		fId, _ := strconv.ParseInt(strFileId, 10, 64)
		gId, _ := strconv.ParseInt(strGroupId, 10, 64)

		records = append(records, ExpireRecord{
			FileID:    fId,
			GroupID:   gId,
			CreatedAt: createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (h *DatabaseService) DeleteByFileIds(fileIds []int64) error {
	if len(fileIds) == 0 {
		return nil
	}

	strFileIds := make([]string, len(fileIds))
	for i, id := range fileIds {
		strFileIds[i] = strconv.FormatInt(id, 10)
	}

	query := `DELETE FROM "ExpireFile" WHERE "fileId" = ANY($1)`

	_, err := h.pool.Exec(context.Background(), query, strFileIds)
	return err
}
