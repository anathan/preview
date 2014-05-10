package common

import (
	"github.com/ngerakines/codederror"
	"log"
)

var (
	ErrorNotImplemented                  = codederror.NewCodedError([]string{"PRV", "COM"}, 1)
	ErrorSourceAssetExpired              = codederror.NewCodedError([]string{"PRV", "COM"}, 2)
	ErrorNoRenderersSupportFileType      = codederror.NewCodedError([]string{"PRV", "COM"}, 3)
	ErrorFileTooLarge                    = codederror.NewCodedError([]string{"PRV", "COM"}, 4)
	ErrorNoDownloadUrlsWork              = codederror.NewCodedError([]string{"PRV", "COM"}, 6)
	ErrorTooLittleWorkRequested          = codederror.NewCodedError([]string{"PRV", "COM"}, 7)
	ErrorUnknownError                    = codederror.NewCodedError([]string{"PRV", "COM"}, 8)
	ErrorUnableToFindGeneratedAssetsById = codederror.NewCodedError([]string{"PRV", "COM"}, 9)
	ErrorNoGeneratedAssetsFoundForId     = codederror.NewCodedError([]string{"PRV", "COM"}, 10)
	ErrorUnableToFindSourceAssetsById    = codederror.NewCodedError([]string{"PRV", "COM"}, 11)
	ErrorNoSourceAssetsFoundForId        = codederror.NewCodedError([]string{"PRV", "COM"}, 12)
	ErrorUnableToFindTemplatesById       = codederror.NewCodedError([]string{"PRV", "COM"}, 13)
	ErrorNoTemplatesFoundForId           = codederror.NewCodedError([]string{"PRV", "COM"}, 14)
	ErrorCouldNotDetermineRenderSize     = codederror.NewCodedError([]string{"PRV", "COM"}, 15)
	ErrorCouldNotResizeImage             = codederror.NewCodedError([]string{"PRV", "COM"}, 16)
	ErrorCouldNotUploadAsset             = codederror.NewCodedError([]string{"PRV", "COM"}, 17)
	ErrorS3FileNotFound                  = codederror.NewCodedError([]string{"PRV", "COM"}, 18)
	ErrorCouldNotDetermineFileSize       = codederror.NewCodedError([]string{"PRV", "COM"}, 19)
	ErrorNoTemplateForId                 = codederror.NewCodedError([]string{"PRV", "COM"}, 20)
	ErrorTemplateHeightAttributeMissing  = codederror.NewCodedError([]string{"PRV", "COM"}, 21)
	ErrorGeneratedAssetCouldNotBeUpdated = codederror.NewCodedError([]string{"PRV", "COM"}, 22)
	ErrorUploaderDoesNotSupportUrl       = codederror.NewCodedError([]string{"PRV", "COM"}, 23)
	ErrorInvalidFileId                   = codederror.NewCodedError([]string{"PRV", "COM"}, 24)
	ErrorMissingFieldType                = codederror.NewCodedError([]string{"PRV", "COM"}, 25)
	ErrorMissingFieldUrl                 = codederror.NewCodedError([]string{"PRV", "COM"}, 26)
	ErrorMissingFieldSize                = codederror.NewCodedError([]string{"PRV", "COM"}, 27)

	AllErrors = []*codederror.CodedError{
		ErrorNotImplemented,
		ErrorSourceAssetExpired,
		ErrorNoRenderersSupportFileType,
		ErrorFileTooLarge,
		ErrorNoDownloadUrlsWork,
		ErrorTooLittleWorkRequested,
		ErrorUnknownError,
		ErrorUnableToFindGeneratedAssetsById,
		ErrorNoGeneratedAssetsFoundForId,
		ErrorUnableToFindSourceAssetsById,
		ErrorNoSourceAssetsFoundForId,
		ErrorUnableToFindTemplatesById,
		ErrorNoTemplatesFoundForId,
		ErrorCouldNotDetermineRenderSize,
		ErrorCouldNotResizeImage,
		ErrorCouldNotUploadAsset,
		ErrorS3FileNotFound,
		ErrorCouldNotDetermineFileSize,
		ErrorNoTemplateForId,
		ErrorTemplateHeightAttributeMissing,
		ErrorGeneratedAssetCouldNotBeUpdated,
		ErrorUploaderDoesNotSupportUrl,
		ErrorInvalidFileId,
		ErrorMissingFieldType,
		ErrorMissingFieldUrl,
		ErrorMissingFieldSize,
	}
)

// NewGeneratedAssetError returns a correctly formatted error string for generated asset storage manager storage.
func NewGeneratedAssetError(err *codederror.CodedError) string {
	return GeneratedAssetStatusFailed + "," + err.Error()
}

// DumpErrors prints out all of the errors contained in AllErrors.
func DumpErrors() {
	for _, bsnError := range AllErrors {
		log.Println(bsnError.Error(), bsnError.Namespaces(), bsnError.Code())
	}
}
