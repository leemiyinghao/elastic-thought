package elasticthought

import (
	"fmt"
	"net/http"

	"github.com/couchbaselabs/cbfs/client"
	"github.com/couchbaselabs/logg"
	"github.com/tleyden/go-couch"
)

// A solver can generate trained models, which ban be used to make predictions
type Solver struct {
	ElasticThoughtDoc
	DatasetId        string `json:"dataset-id"`
	SpecificationUrl string `json:"specification-url" binding:"required"`
}

// Create a new solver.  If you don't use this, you must set the
// embedded ElasticThoughtDoc Type field.
func NewSolver() *Solver {
	return &Solver{
		ElasticThoughtDoc: ElasticThoughtDoc{Type: DOC_TYPE_SOLVER},
	}
}

// Insert into database (only call this if you know it doesn't arleady exist,
// or else you'll end up w/ unwanted dupes)
func (s Solver) Insert(db couch.Database) (*Solver, error) {

	id, _, err := db.Insert(s)
	if err != nil {
		err := fmt.Errorf("Error inserting solver: %v.  Err: %v", s, err)
		return nil, err
	}

	// load dataset object from db (so we have id/rev fields)
	solver := &Solver{}
	err = db.Retrieve(id, solver)
	if err != nil {
		err := fmt.Errorf("Error fetching solver: %v.  Err: %v", id, err)
		return nil, err
	}

	return solver, nil

}

// download contents of solver-spec-url into cbfs://<solver-id>/spec.prototxt
// and update solver object's solver-spec-url with cbfs url
func (s Solver) SaveSpec(db couch.Database, cbfs *cbfsclient.Client) (*Solver, error) {

	// open stream to source url
	url := s.SpecificationUrl
	resp, err := http.Get(url)
	if err != nil {
		errMsg := fmt.Errorf("Error doing GET on: %v.  %v", url, err)
		return nil, errMsg
	}
	defer resp.Body.Close()

	// save to cbfs
	options := cbfsclient.PutOptions{
		ContentType: "text/plain",
	}
	destPath := fmt.Sprintf("%v/spec.prototxt", s.Id)
	if err := cbfs.Put("", destPath, resp.Body, options); err != nil {
		errMsg := fmt.Errorf("Error writing %v to cbfs: %v", destPath, err)
		return nil, errMsg
	}
	logg.LogTo("REST", "Wrote %v to cbfs", destPath)

	// update solver with cbfs url
	cbfsUrl := fmt.Sprintf("cbfs://%v", destPath)
	s.SpecificationUrl = cbfsUrl

	// save
	solver, err := s.Save(db)
	if err != nil {
		return nil, err
	}

	return solver, nil
}

// Saves the solver to the db, returns latest rev
func (s Solver) Save(db couch.Database) (*Solver, error) {

	// TODO: retry if 409 error
	_, err := db.Edit(s)
	if err != nil {
		return nil, err
	}

	// load latest version of dataset to return
	solver := &Solver{}
	err = db.Retrieve(s.Id, solver)
	if err != nil {
		return nil, err
	}

	return solver, nil

}
