package tasks

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/dnephin/buildpipe/config"
	"github.com/fsouza/go-dockerclient"
	//	"github.com/hashicorp/errwrap"
)

// Task is an interface implemented by all tasks
type Task interface {
	Run() error
	Name() string
}

// Task
type baseTask struct {
	name string
	// TODO: move client to an argument of Run() ?
	client *docker.Client
}

func (t *baseTask) Name() string {
	return t.name
}

// TaskCollection is a collection of Task objects
type TaskCollection struct {
	allTasks []Task
	volumes  map[string]*VolumeTask
	images   map[string]*ImageTask
}

func (c *TaskCollection) add(task Task) {
	c.allTasks = append(c.allTasks, task)
	switch typedTask := task.(type) {
	case *VolumeTask:
		c.volumes[task.Name()] = typedTask
	case *ImageTask:
		c.images[task.Name()] = typedTask
	}
}

func (c *TaskCollection) contains(name string) bool {
	for _, task := range c.allTasks {
		if task.Name() == name {
			return true
		}
	}
	return false
}

func newTaskCollection() *TaskCollection {
	return &TaskCollection{
		volumes: make(map[string]*VolumeTask),
		images:  make(map[string]*ImageTask),
	}
}

func prepareTasks(options RunOptions) (*TaskCollection, error) {
	tasks := newTaskCollection()

	var prepare func(resourceNames []string) error
	// TODO: detect cyclic dependencies
	prepare = func(resourceNames []string) error {
		for _, name := range resourceNames {
			if tasks.contains(name) {
				continue
			}

			// TODO: validate this in the config package so that dry-run is
			// possible
			resource, ok := options.Config.Resources[name]
			if !ok {
				return fmt.Errorf("Resource not defined: %s", name)
			}

			task := buildTaskFromResource(taskOptions{
				name:     name,
				client:   options.Client,
				resource: resource,
				tasks:    tasks,
			})

			prepare(resource.Dependencies())
			tasks.add(task)
		}
		return nil
	}

	if err := prepare(options.Pipelines); err != nil {
		return nil, err
	}
	return tasks, nil
}

type taskOptions struct {
	name     string
	client   *docker.Client
	resource config.Resource
	tasks    *TaskCollection
}

func buildTaskFromResource(options taskOptions) Task {
	switch conf := options.resource.(type) {
	case *config.ImageConfig:
		return NewImageTask(options, conf)
	case *config.CommandConfig:
		return NewCommandTask(options, conf)
	case *config.VolumeConfig:
		return NewVolumeTask(options, conf)
	default:
		panic(fmt.Sprintf("Unexpected config type %T", conf))
	}
}

func executeTasks(tasks *TaskCollection) error {
	log.Debug("executing tasks")
	for _, task := range tasks.allTasks {
		if err := task.Run(); err != nil {
			return err
		}
	}
	return nil
}

// RunOptions are the options supported by Run
type RunOptions struct {
	Client    *docker.Client
	Config    *config.Config
	Pipelines []string
}

// Run one or more pipelines
func Run(options RunOptions) error {
	tasks, err := prepareTasks(options)
	if err != nil {
		return err
	}
	return executeTasks(tasks)
}
