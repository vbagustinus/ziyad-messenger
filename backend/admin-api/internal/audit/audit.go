package audit

import (
	"admin-service/internal/db"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type LogEntry struct {
	ID             int64     `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	ActorID        string    `json:"actor_id"`
	ActorUsername  string    `json:"actor_username"`
	Action         string    `json:"action"`
	TargetResource string    `json:"target_resource"`
	Details        string    `json:"details"`
	IPAddress      string    `json:"ip_address"`
}

func Log(actorID, actorUsername, action, targetResource, details, ip string) error {
	_, err := db.DB.Exec(
		`INSERT INTO audit_logs (timestamp, actor_id, actor_username, action, target_resource, details, ip_address) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), actorID, actorUsername, action, targetResource, details, ip,
	)
	return err
}

func LogJSON(actorID, actorUsername, action, targetResource string, details interface{}, ip string) error {
	b, _ := json.Marshal(details)
	return Log(actorID, actorUsername, action, targetResource, string(b), ip)
}

func GetList(offset, limit int, actorID, action string) ([]LogEntry, error) {
	q := `SELECT id, timestamp, actor_id, actor_username, action, target_resource, details, ip_address FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	if actorID != "" {
		q += ` AND actor_id = ?`
		args = append(args, actorID)
	}
	if action != "" {
		q += ` AND action = ?`
		args = append(args, action)
	}
	q += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []LogEntry{}
	for rows.Next() {
		var e LogEntry
		var ts int64
		err := rows.Scan(&e.ID, &ts, &e.ActorID, &e.ActorUsername, &e.Action, &e.TargetResource, &e.Details, &e.IPAddress)
		if err != nil {
			return nil, err
		}
		e.Timestamp = time.Unix(ts, 0)
		list = append(list, e)
	}
	return list, nil
}

func MustUUID() string { return uuid.New().String() }
