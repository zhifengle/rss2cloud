package p115

import (
	"context"
	"fmt"
	"time"

	"github.com/deadblue/elevengo"
	"github.com/deadblue/elevengo/option"
	"golang.org/x/time/rate"
)

const (
	FileListLimit = 32
)

var limiter *rate.Limiter = rate.NewLimiter(rate.Limit(0.75), 1)

// flatten files, copy from targetDir to newDir
func (ag *Agent) MoveFlattenFiles(targetDirId string, parentDirId string, newDirName string) error {
	if targetDirId == "" {
		return fmt.Errorf("targetDirId is empty")
	}
	var targetFile elevengo.File
	if parentDirId == "" {
		ag.Agent.FileGet(targetDirId, &targetFile)
		limiter.Wait(context.Background())
		parentDirId = targetFile.ParentId
		if newDirName == "" {
			newDirName = targetFile.Name + "_flatten"
		}
	}
	// Step 1: Create a new directory
	newDirId, err := ag.Agent.DirMake(parentDirId, newDirName)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	limiter.Wait(context.Background())

	// Step 2: Iterate files in the target directory
	it, err := ag.Agent.FileIterate(targetDirId)
	if err != nil {
		return fmt.Errorf("failed to iterate files: %w", err)
	}
	limiter.Wait(context.Background())

	var fileIds []string

	for i, file := range it.Items() {
		if file.FileId == newDirId {
			continue
		}
		if file.IsDirectory {
			fi, err := ag.Agent.FileIterate(file.FileId)
			limiter.Wait(context.Background())
			if err != nil {
				// return fmt.Errorf("failed to iterate sub folder: %w", err)
				// @TODO ignore error
				continue
			}
			for _, f := range fi.Items() {
				fileIds = append(fileIds, f.FileId)
			}
		} else {
			fileIds = append(fileIds, file.FileId)
		}
		if i%FileListLimit == 0 {
			limiter.Wait(context.Background())
		}
	}

	// Step 4: Move files to the new directory

	for i, ids := range chunkBy(fileIds, 40) {
		if err := ag.Agent.FileMove(newDirId, ids); err != nil {
			return fmt.Errorf("failed to move files: %w", err)
		}
		limiter.Wait(context.Background())
		if i != len(fileIds)/40 {
			time.Sleep(time.Second * time.Duration(chunkDelay))
		}
	}

	// step 5: delete empty dir
	// if err := ag.RemoveEmptyDir(targetDirId); err != nil {
	// 	return fmt.Errorf("failed to delete directory: %w", err)
	// }

	return nil
}

// remove empty dir in a dir
func (ag *Agent) RemoveEmptyDir(dirId string) error {
	it, err := ag.Agent.FileIterate(dirId)
	if err != nil {
		return fmt.Errorf("failed to iterate files: %w", err)
	}
	limiter.Wait(context.Background())
	var fileIds []string
	for i, file := range it.Items() {
		if file.IsDirectory {
			fi, err := ag.Agent.FileIterate(file.FileId)
			if err != nil {
				return fmt.Errorf("failed to iterate files: %w", err)
			}
			limiter.Wait(context.Background())
			if fi.Count() == 0 {
				fileIds = append(fileIds, file.FileId)
			}
		}
		if i%FileListLimit == 0 {
			if err := ag.Agent.FileDelete(fileIds); err != nil {
				return fmt.Errorf("failed to delete files: %w", err)
			}
			fileIds = nil
			limiter.Wait(context.Background())
		}
	}
	// delete empty dir
	for _, ids := range chunkBy(fileIds, 40) {
		if err := ag.Agent.FileDelete(ids); err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
		limiter.Wait(context.Background())
	}

	return nil
}

// search file in dir and move to new dir
func (ag *Agent) SearchAndMoveFiles(targetDirId string, distDirId string, keyword string, fileType int) error {
	if targetDirId == "" || distDirId == "" {
		return fmt.Errorf("targetDirId or distDirId is empty")
	}

	fileOpt := &option.FileListOptions{Type: fileType, ExtName: ""}
	it, err := ag.Agent.FileSearch(targetDirId, keyword, fileOpt)
	if err != nil {
		return fmt.Errorf("failed to search files: %w", err)
	}
	limiter.Wait(context.Background())

	var fileIds []string
	for i, file := range it.Items() {
		if file.IsDirectory {
			continue
		}
		if i%FileListLimit == 0 {
			limiter.Wait(context.Background())
		}
		fileIds = append(fileIds, file.FileId)
	}
	size := 40
	for i, ids := range chunkBy(fileIds, size) {
		if err := ag.Agent.FileMove(distDirId, ids); err != nil {
			return fmt.Errorf("failed to move files: %w", err)
		}
		limiter.Wait(context.Background())
		if i != len(fileIds)/size {
			time.Sleep(time.Second * time.Duration(chunkDelay))
		}
	}

	return nil
}
