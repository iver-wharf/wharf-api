package main

import (
	"io"
	"math"

	v1 "github.com/iver-wharf/wharf-api/v5/api/wharfapi/v1"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type grpcWharfServer struct {
	v1.UnimplementedBuildsServer
	db *gorm.DB
}

func (s *grpcWharfServer) CreateLogStream(stream v1.Builds_CreateLogStreamServer) error {
	var logsInserted uint64
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			stream.SendAndClose(&v1.CreateLogStreamResponse{
				LinesInserted: logsInserted,
			})
		} else if err != nil {
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
				Message:   line.Message,
				Timestamp: line.Timestamp.AsTime(),
			}
		}
		if len(dbLogs) == 0 {
			continue
		}
		createdLogs, err := createLogBatch(s.db.WithContext(stream.Context()), dbLogs)
		if err != nil {
			return status.Errorf(codes.Internal, "insert logs: %v", err)
		}
		for _, dbLog := range createdLogs {
			build(dbLog.BuildID).Submit(response.Log{
				LogID:     dbLog.LogID,
				BuildID:   dbLog.BuildID,
				Message:   dbLog.Message,
				Timestamp: dbLog.Timestamp,
			})
		}
		logsInserted += uint64(len(createdLogs))
	}
}
