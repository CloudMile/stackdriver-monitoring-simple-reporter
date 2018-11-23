package gcp

import (
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"log"

	"google.golang.org/api/cloudresourcemanager/v1beta1"
)

func GetProjects(ctx context.Context) []string {
	client, err := google.DefaultClient(ctx, cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		log.Fatal("SetContext: ", err.Error())
	}

	svc, err := cloudresourcemanager.New(client)
	if err != nil {
		log.Fatal("GetProjects: ", err.Error())
	}

	projectsListCall := svc.Projects.List()
	listResp, err := projectsListCall.Do()
	if err != nil {
		log.Fatal("GetProjects: ", err.Error())
	}

	projects := listResp.Projects
	projectIDs := make([]string, len(projects))
	for i := range projects {
		projectIDs[i] = projects[i].ProjectId
	}

	return projectIDs
}
