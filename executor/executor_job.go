package executor

import (
	"context"
	"fmt"

	"dagger.io/dagger"

	"github.com/aweris/gale/gha"
	runnerpkg "github.com/aweris/gale/runner"
)

type JobExecutor struct {
	client        *dagger.Client
	runner        *runnerpkg.Runner
	workflow      *gha.Workflow
	job           *gha.Job
	context       *gha.RunContext
	stepExecutors []StepExecutor
}

// NewJobExecutor creates a new job executor.
func NewJobExecutor(ctx context.Context, client *dagger.Client, workflow *gha.Workflow, job *gha.Job, context *gha.RunContext) (*JobExecutor, error) {
	// Create runner
	runner, err := runnerpkg.NewRunner(ctx, client)
	if err != nil {
		return nil, err
	}

	return &JobExecutor{
		client:        client,
		runner:        runner,
		workflow:      workflow,
		job:           job,
		context:       context,
		stepExecutors: []StepExecutor{},
	}, nil
}

func (j *JobExecutor) Execute(ctx context.Context) error {
	if err := j.setup(ctx); err != nil {
		return err
	}

	for _, se := range j.stepExecutors {
		if err := se.pre(ctx, j.runner); err != nil {
			return err
		}
	}

	for _, se := range j.stepExecutors {
		if err := se.main(ctx, j.runner); err != nil {
			return err
		}
	}

	for _, se := range j.stepExecutors {
		if err := se.post(ctx, j.runner); err != nil {
			return err
		}
	}

	return nil
}

func (j *JobExecutor) setup(ctx context.Context) error {
	fmt.Println("Set up job")

	// TODO: this is a hack, we should find better way to do this
	j.runner.WithExec("mkdir", "-p", j.context.Github.Workspace)

	j.runner.WithEnvironment(j.context.ToEnv())
	j.runner.WithEnvironment(j.workflow.Environment)
	j.runner.WithEnvironment(j.job.Environment)

	for _, step := range j.job.Steps {
		action, err := gha.LoadActionFromSource(ctx, j.client, step.Uses)
		if err != nil {
			return err
		}

		path := j.runner.WithTempDirectory(action.Directory)

		j.stepExecutors = append(j.stepExecutors, NewStepActionExecutor(step, action, path, j.context.ToEnv(), j.workflow.Environment, j.job.Environment))

		fmt.Printf("Download action repository '%s'\n", step.Uses)
	}

	return nil
}