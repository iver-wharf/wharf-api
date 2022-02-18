package main

import (
	"math"

	v1 "github.com/iver-wharf/wharf-api/v5/api/wharfapi/v1"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"gorm.io/gorm"
)

type grpcWharfServer struct {
	v1.UnimplementedBuildsServer
	db *gorm.DB
}

func (s *grpcWharfServer) CreateLogStream(stream v1.Builds_CreateLogStreamServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		dbLogs := make([]database.Log, len(msg.Lines))
		for i, line := range msg.Lines {
			if err := line.Timestamp.CheckValid(); err != nil {
				log.Warn().WithError(err).
					Message("Received invalid log timestamp, skipping.")
				continue
			}
			if line.BuildId > math.MaxUint {
				log.Warn().WithUint64("buildId", line.BuildId).
					Message("Received too big log build ID, skipping.")
				continue
			}
			dbLogs[i] = database.Log{
				BuildID:   uint(line.BuildId),
				Message:   line.Line,
				Timestamp: line.Timestamp.AsTime(),
			}
		}
		if len(dbLogs) == 0 {
			continue
		}
		// TODO: Join on build table to ignore non-existing build_id's
		// TODO: Publish logs on logs broadcast channels
		s.db.WithContext(stream.Context()).
			Create(dbLogs)
	}
}
