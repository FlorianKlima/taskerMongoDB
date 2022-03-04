package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/gookit/color.v1"
	"log"
	"os"
	"time"
)

var collection *mongo.Collection
var ctx = context.TODO()

func init() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	collection = client.Database("tasker").Collection("tasks")
}

type Task struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Title     string             `bson:"title"`
	Done      bool               `bson:"done"`
}

func main() {
	app := &cli.App{
		Name:  "Tasker",
		Usage: "A simple task manager",
		Action: func(c *cli.Context) error {
			tasks, err := getPending()
			if err != nil {
				if err == mongo.ErrNoDocuments {
					fmt.Print("No tasks. Run add")
					return nil
				}
				return err
			}
			printTasks(tasks)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add a task to the List",
				Action: func(c *cli.Context) error {
					str := c.Args().First()
					if str == "" {
						log.Fatal("Please enter a task")
					}
					task := &Task{
						ID:        primitive.NewObjectID(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Title:     str,
						Done:      false,
					}
					return createTask(task)
				},
			},
			{
				Name:    "all",
				Aliases: []string{"l"},
				Usage:   "List all tasks",
				Action: func(c *cli.Context) error {
					tasks, err := getAll()
					if err != nil {
						if err == mongo.ErrNoDocuments {
							log.Println("No tasks found")
							return nil
						}
						return err
					}
					printTasks(tasks)
					return nil
				},
			},
			{
				Name:    "done",
				Aliases: []string{"d"},
				Usage:   "complete a task on the list",
				Action: func(c *cli.Context) error {
					title := c.Args().First()
					return completeTask(title)
				},
			},
			{
				Name:    "finished",
				Aliases: []string{"f"},
				Usage:   "List finished tasks",
				Action: func(c *cli.Context) error {
					tasks, err := getFinished()
					if err != nil {
						if err == mongo.ErrNoDocuments {
							log.Println("No tasks found")
							return nil
						}
						return err
					}
					printTasks(tasks)
					return nil
				},
			},
			{
				Name:  "rm",
				Usage: "Delete a task",
				Action: func(c *cli.Context) error {
					title := c.Args().First()
					err := deleteTask(title)
					if err != nil {
						return err
					}
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func createTask(task *Task) error {
	_, err := collection.InsertOne(ctx, task)
	return err
}
func getAll() ([]*Task, error) {
	filter := bson.D{}
	return filterTasks(filter)
}
func filterTasks(filter interface{}) ([]*Task, error) {
	var tasks []*Task
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		return tasks, err
	}
	for cur.Next(ctx) {
		var t Task
		err := cur.Decode(&t)
		if err != nil {
			return tasks, err
		}
		tasks = append(tasks, &t)
	}
	if err := cur.Err(); err != nil {
		return tasks, err
	}
	cur.Close(ctx)
	if len(tasks) == 0 {
		return tasks, mongo.ErrNoDocuments
	}
	return tasks, nil
}
func printTasks(tasks []*Task) {
	for i, v := range tasks {
		if v.Done {
			color.Green.Printf("%d: %s\n", i+1, v.Title)
		} else {
			color.BgRed.Printf("%d: %s\n", i+1, v.Title)
		}
	}
}
func completeTask(title string) error {
	filter := bson.D{primitive.E{Key: "title", Value: title}}
	update := bson.D{primitive.E{Key: "$set", Value: bson.D{primitive.E{Key: "done", Value: true}}}}
	t := &Task{}
	return collection.FindOneAndUpdate(ctx, filter, update).Decode(t)
}
func getPending() ([]*Task, error) {
	filter := bson.D{
		primitive.E{Key: "done", Value: false},
	}
	return filterTasks(filter)
}
func getFinished() ([]*Task, error) {
	filter := bson.D{
		primitive.E{Key: "done", Value: true},
	}
	return filterTasks(filter)
}
func deleteTask(title string) error {
	filter := bson.D{primitive.E{Key: "title", Value: title}}
	res, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("No tasks were deleted")
	}
	return nil
}
