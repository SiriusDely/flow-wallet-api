package jobs

import (
	"fmt"
	"net/http"

	"github.com/eqlabs/flow-wallet-service/datastore"
	"github.com/eqlabs/flow-wallet-service/errors"
	"github.com/google/uuid"
)

// Service defines the API for job HTTP handlers.
type Service struct {
	db Store
}

// NewService initiates a new job service.
func NewService(db Store) *Service {
	return &Service{db}
}

// List returns all jobs in the datastore.
func (s *Service) List(limit, offset int) (result []Job, err error) {
	o := datastore.ParseListOptions(limit, offset)
	return s.db.Jobs(o)
}

// Details returns a specific job.
func (s *Service) Details(jobId string) (result Job, err error) {
	id, err := uuid.Parse(jobId)
	if err != nil {
		// Convert error to a 400 RequestError
		err = &errors.RequestError{
			StatusCode: http.StatusBadRequest,
			Err:        fmt.Errorf("invalid job id"),
		}
		return
	}

	// Get from datastore
	result, err = s.db.Job(id)
	if err != nil && err.Error() == "record not found" {
		// Convert error to a 404 RequestError
		err = &errors.RequestError{
			StatusCode: http.StatusNotFound,
			Err:        fmt.Errorf("job not found"),
		}
		return
	}

	return
}