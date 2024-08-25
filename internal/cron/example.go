package cron

import (
	"context"
	"fmt"
	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/fxcron"
)

type ExampleCronJob struct {
	config *config.Config
}

func NewExampleCronJob(config *config.Config) *ExampleCronJob {
	return &ExampleCronJob{
		config: config,
	}
}

func (c *ExampleCronJob) Name() string {
	return "example-cron-job"
}

func (c *ExampleCronJob) Run(ctx context.Context) error {

	rawurl := "sftp://zdop_it2:IT2.0@2024@data.cept.gov.in"
	s, conn := connectSftp(rawurl)
	defer conn.Close()
	defer s.Close()

	localSourceDirUpload := "./files"
	localDestinationDirUpload := "./files/archive"
	RemoteDirUpload := "/IT2/TO_CSI/"

	localDestinationDownload := "./downloads/"
	RemoteDestinationDownload := "/IT2/TO_CSI/"

	files, err := listLocalFiles(localSourceDirUpload)
	if err != nil {
		fmt.Println("Error in listing files", err)
	}

	//move file from one folder to other locally

	for _, f := range files {
		ff, _ := f.Info()
		//fmt.Println("Files are :", ff)
		localfile := localSourceDirUpload + "/" + ff.Name()
		remotefile := RemoteDirUpload + ff.Name()
		err := uploadFile(s, localfile, remotefile)
		if err != nil {
			fmt.Println("Came inside list files error")
			fmt.Println(err)
		}
		err = moveLocalFile(localSourceDirUpload, localDestinationDirUpload, ff.Name())
		if err != nil {
			fmt.Println("Error moving file locally", err)
		}
		//fmt.Println("file succesfully moved")

	}

	remotefiles, err := listSftpFiles(s, RemoteDestinationDownload)
	if err != nil {
		fmt.Println("Error list files", err)
	}
	//For the list files from sftp loop through download files and move files to archive
	for _, f := range remotefiles {
		file := f.Name()
		err = downloadFile(s, RemoteDestinationDownload+file, localDestinationDownload+file)
		if err != nil {
			fmt.Println("Came inside list files error")
			fmt.Println(err)
		}

	}
	// err := uploadFile(s, filename, "/IT2/TO_CSI/"+filename)
	// if err != nil {
	// 	fmt.Println("Came inside list files error")
	// 	fmt.Println(err)
	// }

	// defer s.Close()
	// contextual job name and execution id
	name, id := fxcron.CtxCronJobName(ctx), fxcron.CtxCronJobExecutionId(ctx)

	

	c.config.AppName()
	// contextual logging
	fxcron.CtxLogger(ctx).Info().Msgf("example log from app:%s, job:%s, id:%s", c.config.AppName(), name, id)
	//fmt.Println(s)
	// returned errors will automatically be logged
	return nil
}
