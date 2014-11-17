package elasticthought

import (
	"archive/tar"
	"bytes"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/couchbaselabs/go.assert"
	"github.com/couchbaselabs/logg"
)

func TestTransform(t *testing.T) {

	dataset := Dataset{
		TrainingDataset: TrainingDataset{
			SplitPercentage: 0.5,
		},
		TestDataset: TestDataset{
			SplitPercentage: 0.5,
		},
	}

	splitter := DatasetSplitter{
		Dataset: dataset,
	}

	// Create a test tar archive
	buf := new(bytes.Buffer)
	createTestArchive(buf)

	// Open the tar archive for reading.
	r := bytes.NewReader(buf.Bytes())
	tr := tar.NewReader(r)

	// create two writers
	bufTrain := new(bytes.Buffer)
	bufTest := new(bytes.Buffer)
	twTrain := tar.NewWriter(bufTrain)
	twTest := tar.NewWriter(bufTest)

	// pass these into transform
	err := splitter.transform(tr, twTrain, twTest)
	assert.True(t, err == nil)

	// assert that the both the training and test tar archives
	// have been split 50/50 over each "label" folder, so each "label"
	// folder should now have one example each.  Ie, one "foo/x.txt" and
	// one "bar/x.txt".  Also, the resulting split sets must be disjoint.
	buffers := []*bytes.Buffer{bufTrain, bufTest}
	for _, buffer := range buffers {
		readerVerify := bytes.NewReader(buffer.Bytes())
		trVerify := tar.NewReader(readerVerify)

		seenFoos := map[string]string{}
		seenBars := map[string]string{}
		numFoo := 0
		numBar := 0
		for {
			hdr, err := trVerify.Next()
			if err == io.EOF {
				// end of tar archive
				break
			}
			if err != nil {
				log.Fatalln(err)
			}
			if strings.HasPrefix(hdr.Name, "foo") {
				numFoo += 1
				// make sure it's the first time seeing this filename
				if _, ok := seenFoos[hdr.Name]; ok {
					logg.LogPanic("Not first time seeing: %v", hdr.Name)
				}
				seenFoos[hdr.Name] = hdr.Name
			}
			if strings.HasPrefix(hdr.Name, "bar") {
				numBar += 1
				if _, ok := seenBars[hdr.Name]; ok {
					logg.LogPanic("Not first time seeing: %v", hdr.Name)
				}
				seenBars[hdr.Name] = hdr.Name

			}

		}

		assert.Equals(t, numFoo, 1)
		assert.Equals(t, numBar, 1)

	}

}

func TestValidateValid(t *testing.T) {

	buf := new(bytes.Buffer)
	var files = []tarFile{
		{"foo/1.txt", "Hello 1."},
		{"foo/2.txt", "Hello 2."},
		{"bar/1.txt", "Hello bar 1."},
		{"bar/2.txt", "Hello bar 2."},
	}
	createArchive(buf, files)
	reader := bytes.NewReader(buf.Bytes())
	tr := tar.NewReader(reader)

	splitter := DatasetSplitter{}
	ok, err := splitter.validate(tr)
	assert.True(t, ok)
	assert.True(t, err == nil)

}

func TestValidateTooDeep(t *testing.T) {

	buf := new(bytes.Buffer)
	var files = []tarFile{
		{"a/foo/1.txt", "Hello 1."},
		{"a/bar/1.txt", "Hello bar 1."},
	}
	createArchive(buf, files)
	reader := bytes.NewReader(buf.Bytes())
	tr := tar.NewReader(reader)

	splitter := DatasetSplitter{}
	ok, err := splitter.validate(tr)
	assert.False(t, ok)
	assert.True(t, err != nil)

}

func TestCreateMap(t *testing.T) {

	buf := new(bytes.Buffer)
	var files = []tarFile{
		{"foo/1.txt", "Hello 1."},
		{"bar/1.txt", "Hello bar 1."},
		{"bar/2.txt", "Hello bar 2."},
	}
	createArchive(buf, files)
	reader := bytes.NewReader(buf.Bytes())
	tr := tar.NewReader(reader)

	splitter := DatasetSplitter{}
	filemap, err := splitter.createMap(tr)
	assert.True(t, err == nil)

	assert.Equals(t, len(filemap["foo"]), 1)
	assert.Equals(t, len(filemap["bar"]), 2)

}

type tarFile struct {
	Name string
	Body string
}

func createTestArchive(buf *bytes.Buffer) {

	var files = []tarFile{
		{"foo/1.txt", "Hello 1."},
		{"foo/2.txt", "Hello 2."},
		{"bar/1.txt", "Hello bar 1."},
		{"bar/2.txt", "Hello bar 2."},
	}
	createArchive(buf, files)

}

func createArchive(buf *bytes.Buffer, tarFiles []tarFile) {

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	for _, file := range tarFiles {
		hdr := &tar.Header{
			Name: file.Name,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatalln(err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			log.Fatalln(err)
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		log.Fatalln(err)
	}

}
