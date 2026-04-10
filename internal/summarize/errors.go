package summarize

import "errors"

// ErrVectorIndexAfterFileWrite is returned when the summary file was written but embedding/vector upsert failed (EP-002 REQ-02.016).
var ErrVectorIndexAfterFileWrite = errors.New("summarize: vector indexing failed after summary file was written")
