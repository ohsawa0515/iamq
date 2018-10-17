package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3"
)

const (
	timeout = 30
)

var (
	q string
)

// AwsUser --
type AwsUser struct {
	UserId           string     `db:"UserId"`
	UserName         string     `db:"UserName"`
	Path             string     `db:"Path"`
	Arn              string     `db:"Arn"`
	CreateDate       *time.Time `db:"CreateDate"`
	PasswordLastUsed *time.Time `db:"PasswordLastUsed"`
}

// Client --
type Client struct {
	awsSvc iamiface.IAMAPI
	db     *sqlx.DB
}

// NewClient --
func NewClient(iamSvc iamiface.IAMAPI, sourceName string) (*Client, error) {
	db, err := sqlx.Connect("sqlite3", sourceName)
	if err != nil {
		return nil, err
	}
	return &Client{
		awsSvc: iamSvc,
		db:     db,
	}, nil
}

func (c *Client) InsertAwsUsers(ctx context.Context) error {
	resp, err := c.awsSvc.ListUsersWithContext(ctx, &iam.ListUsersInput{})
	if err != nil {
		return err
	}

	tx := c.db.MustBegin()
	for _, user := range resp.Users {
		tx.MustExec("INSERT INTO AwsUser (UserId, UserName, Path, Arn, CreateDate, PasswordLastUsed) VALUES (?, ?, ?, ?, ?, ?)", user.UserId, user.UserName, user.Path, user.Arn, user.CreateDate, user.PasswordLastUsed)
	}
	tx.Commit()

	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	sess := session.Must(session.NewSession())
	client, err := NewClient(iam.New(sess), "./iam.sqlite3")
	if err != nil {
		log.Fatal(err)
	}

	// AWS Users
	q = `
		DROP TABLE IF EXISTS AwsUser;
		CREATE TABLE AwsUser (
		UserId VARCHAR(21) PRIMARY KEY,
		UserName VARCHAR(63) NOT NULL,
		Path VARCHAR(63) NOT NULL,
		Arn VARCHAR(255) NOT NULL,
		CreateDate DATETIME NOT NULL,
		PasswordLastUsed DATETIME
	   );`
	client.db.MustExec(q)
	if err := client.InsertAwsUsers(ctx); err != nil {
		log.Fatal(err)
	}
}
