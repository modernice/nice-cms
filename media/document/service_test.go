package document_test

import (
	_ "embed"
)

//go:embed testdata/example.pdf
var examplePDF []byte

var (
	exampleDocumentName = "example-doc"
	exampleName         = "Example"
	exampleDisk         = "foo"
	examplePath         = "/example/example.pdf"
)

// func TestService_Upload(t *testing.T) {
// 	docs := document.InMemoryRepository()
// 	svc := document.NewService(docs)

// 	r := bytes.NewReader(examplePDF)

// 	doc, err := svc.Upload(context.Background(), r, exampleDocumentName, exampleName, exampleDisk, examplePath)
// 	if err != nil {
// 		t.Fatalf("upload failed: %v", err)
// 	}

// 	wantDoc := media.NewDocument(exampleDocumentName, exampleName, exampleDisk, examplePath, len(examplePDF))

// 	if !reflect.DeepEqual(wantDoc, doc) {
// 		t.Fatalf("Upload returned wrong Document. want=%v got=%v", wantDoc, doc)
// 	}
// }
