package common

import (
	"github.com/ngerakines/codederror"
	"log"
)

var (
	ErrorNotImplemented                  = codederror.NewCodedError([]string{"PRV", "COM"}, 1, "Something wasn't implemented")
	ErrorSourceAssetExpired              = codederror.NewCodedError([]string{"PRV", "COM"}, 2, "The source asset has expired.")
	ErrorNoRenderersSupportFileType      = codederror.NewCodedError([]string{"PRV", "COM"}, 3, "No renderers support the given file type.")
	ErrorFileTooLarge                    = codederror.NewCodedError([]string{"PRV", "COM"}, 4, "The file is too large.")
	ErrorNoDownloadUrlsWork              = codederror.NewCodedError([]string{"PRV", "COM"}, 6, "No download urls work.")
	ErrorTooLittleWorkRequested          = codederror.NewCodedError([]string{"PRV", "COM"}, 7, "Too little work requested")
	ErrorUnknownError                    = codederror.NewCodedError([]string{"PRV", "COM"}, 8, "Something mysterious happened.")
	ErrorUnableToFindGeneratedAssetsById = codederror.NewCodedError([]string{"PRV", "COM"}, 9, "No generated assets for the id are found.")
	ErrorNoGeneratedAssetsFoundForId     = codederror.NewCodedError([]string{"PRV", "COM"}, 10, "No generated assets for the id are found.")
	ErrorUnableToFindSourceAssetsById    = codederror.NewCodedError([]string{"PRV", "COM"}, 11, "No source assets for the id are found.")
	ErrorNoSourceAssetsFoundForId        = codederror.NewCodedError([]string{"PRV", "COM"}, 12, "No source assets for the id are found.")
	ErrorUnableToFindTemplatesById       = codederror.NewCodedError([]string{"PRV", "COM"}, 13, "No templates for the id are found.")
	ErrorNoTemplatesFoundForId           = codederror.NewCodedError([]string{"PRV", "COM"}, 14, "No templates for the id are found.")
	ErrorCouldNotDetermineRenderSize     = codederror.NewCodedError([]string{"PRV", "COM"}, 15, "Could not determine the size of the render.")
	ErrorCouldNotResizeImage             = codederror.NewCodedError([]string{"PRV", "COM"}, 16, "Could not resize image.")
	ErrorCouldNotUploadAsset             = codederror.NewCodedError([]string{"PRV", "COM"}, 17, "Could not upload asset.")
	ErrorS3FileNotFound                  = codederror.NewCodedError([]string{"PRV", "COM"}, 18, "S3 file not found")
	ErrorCouldNotDetermineFileSize       = codederror.NewCodedError([]string{"PRV", "COM"}, 19, "Could not determine size of file.")
	ErrorNoTemplateForId                 = codederror.NewCodedError([]string{"PRV", "COM"}, 20, "No templates for the id are found.")
	ErrorTemplateHeightAttributeMissing  = codederror.NewCodedError([]string{"PRV", "COM"}, 21, "Template is missing required height attribute.")
	ErrorGeneratedAssetCouldNotBeUpdated = codederror.NewCodedError([]string{"PRV", "COM"}, 22, "Generated asset could not be updated.")
	ErrorUploaderDoesNotSupportUrl       = codederror.NewCodedError([]string{"PRV", "COM"}, 23, "Uploader does not support protocol.")
	ErrorInvalidFileId                   = codederror.NewCodedError([]string{"PRV", "COM"}, 24, "Invalid file id.")
	ErrorMissingFieldType                = codederror.NewCodedError([]string{"PRV", "COM"}, 25, "Missing type field.")
	ErrorMissingFieldUrl                 = codederror.NewCodedError([]string{"PRV", "COM"}, 26, "Missing url field.")
	ErrorMissingFieldSize                = codederror.NewCodedError([]string{"PRV", "COM"}, 27, "Missing size field.")
	ErrorCouldNotDetermineFileType       = codederror.NewCodedError([]string{"PRV", "COM"}, 28, "Could not determine type of file.")

	AllErrors = []codederror.CodedError{
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
func NewGeneratedAssetError(err codederror.CodedError) string {
	return GeneratedAssetStatusFailed + "," + err.Error()
}

// DumpErrors prints out all of the errors contained in AllErrors.
func DumpErrors() {
	for _, bsnError := range AllErrors {
		log.Println(bsnError.Error(), bsnError.Namespaces(), bsnError.Code(), bsnError.Description())
	}
}
