package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type DatabaseService struct {
	conn *pgx.Conn
}

func NewDatabaseService(databaseUrl string) *DatabaseService {
	conn, err := pgx.Connect(context.Background(), databaseUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	return &DatabaseService{
		conn: conn,
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

	err := h.conn.QueryRow(context.Background(), query, strFileId, strGroupId).Scan(&createdAt)
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
		WHERE "createdAt" <= NOW() - INTERVAL '12 hours'
		LIMIT 100`

	rows, err := h.conn.Query(context.Background(), query)
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

	_, err := h.conn.Exec(context.Background(), query, strFileIds)
	return err
}
