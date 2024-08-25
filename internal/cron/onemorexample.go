package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"	
	"github.com/jackc/pgx/v5"
	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/fxcron"	
	"github.com/templatedop/ftptemplate/db"
	"github.com/templatedop/ftptemplate/repo"
)

type OneExampleCronJob struct {
	config *config.Config
	db     *db.DB	
}

func OneNewExampleCronJob(config *config.Config, db *db.DB) *OneExampleCronJob {
	return &OneExampleCronJob{
		config: config,
		db:     db,
	}
}

func (c *OneExampleCronJob) Name() string {
	return "example-cron-job"
}

func filecreate(data interface{}, f string) (filename string, err error) {

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	file, err := os.Create(f)
	if err != nil {

		fmt.Println("Error creating file:", err)
		return "", err
	}
	defer file.Close()
	_, err = file.Write(jsonData)
	if err != nil {

		//log.Error("Error writing to file:",err)
		fmt.Println("Error writing to file:", err)
		return "", err
	}

	//log.Info("File created successfully")
	fmt.Println("File created successfully")

	return file.Name(), nil

}

type Data struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type User struct {
	ID    uint64 `json:"id" db:"id" select:"-" `
	Name  string `json:"name" insert:"name" select:"name" insert_pickup:"name"`
	Email string `json:"email" insert:"email" select:"email"`
}

func (u User) String() string {
	return fmt.Sprintf("%d|%s|%s", u.ID, u.Name, u.Email)
}

type Stringer interface {
	String() string
}

func WriteSliceToFile[T Stringer](data []T, filename string) error {
	// Create or open the file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Iterate over the slice and write each item to the file using the String method
	for _, item := range data {
		_, err = file.WriteString(item.String() + "\n")
		if err != nil {
			return fmt.Errorf("error writing to file: %v", err)
		}
	}

	fmt.Println("Data written to file successfully.")
	return nil
}

func (c *OneExampleCronJob) Run(ctx context.Context) error {
	
	query := repo.Psql.Select("id,name,email").From("users")
	fmt.Println("Query is", query)
	d, e := repo.SelectRows(context.Background(), c.db, query, pgx.RowToAddrOfStructByPos[User])

	if e != nil {
		fmt.Println("Error in selecting rows", e)
		return e
	}
	
	filename := "./files/users_slice.txt"
	err := WriteSliceToFile(d, filename)
	if err != nil {
		fmt.Println("Error writing slice data to file:", err)
		return err
	}
	
	name, id := fxcron.CtxCronJobName(ctx), fxcron.CtxCronJobExecutionId(ctx)

	c.config.AppName()
	// contextual logging
	fxcron.CtxLogger(ctx).Info().Msgf("It's from one example log from app:%s, job:%s, id:%s", c.config.AppName(), name, id)

	// returned errors will automatically be logged
	return nil
}
