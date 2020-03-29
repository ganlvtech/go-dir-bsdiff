package patch

import "fmt"

const (
	BsDiffFileSuffix = ".bsdiff"
	ManifestFileName = "patch.json"
)

const (
	OperationTypeCopyOld = "copy"
	OperationTypeCopyNew = "new"
	OperationTypePatch   = "patch"
)

type Manifest struct {
	ManifestVersion string            `json:"manifest_version"`
	BulkSize        int               `json:"bulk_size"`
	OldMd5          map[string]string `json:"old_md5"`
	NewMd5          map[string]string `json:"new_md5"`
	Patches         map[string]string `json:"patches"`
}

func NewPatchManifest(bulkSize int) *Manifest {
	return &Manifest{
		ManifestVersion: "1.0",
		BulkSize:        bulkSize,
		OldMd5:          nil,
		NewMd5:          nil,
		Patches:         nil,
	}
}

func GetPartNewFileName(basePath string, partIndex int) string {
	return fmt.Sprintf("%s.part.%d", basePath, partIndex)
}

func GetPartDiffFileName(basePath string, partIndex int) string {
	return fmt.Sprintf("%s.part.%d", basePath, partIndex) + BsDiffFileSuffix
}
