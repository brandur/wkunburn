package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/brandur/wanikaniapi"
)

const (
	maxTargets = 100
)

func main() {
	rand.Seed(time.Now().UnixNano())

	logger := &wanikaniapi.LeveledLogger{Level: wanikaniapi.LevelDebug}

	apiToken := os.Getenv("WANI_KANI_API_TOKEN")
	if apiToken == "" {
		abort("need WANI_KANI_API_TOKEN")
	}

	numTargets := 2
	if numTargets >= maxTargets {
		abort("number or targets to unburn must be smaller or equal to %v", maxTargets)
	}

	client := wanikaniapi.NewClient(&wanikaniapi.ClientConfig{
		APIToken: apiToken,
		Logger:   logger,
	})

	var assignments []*wanikaniapi.Assignment
	err := client.PageFully(func(id *wanikaniapi.WKID) (*wanikaniapi.PageObject, error) {
		page, err := client.AssignmentList(&wanikaniapi.AssignmentListParams{
			ListParams: wanikaniapi.ListParams{
				PageAfterID: id,
			},
			Burned: wanikaniapi.Bool(true),
		})
		if err != nil {
			return nil, err
		}

		assignments = append(assignments, page.Data...)
		return &page.PageObject, nil
	})
	if err != nil {
		panic(err)
	}

	logger.Infof("Got %v assignment(s)", len(assignments))

	var randomAssignments []*wanikaniapi.Assignment

	if len(assignments) <= numTargets {
		randomAssignments = assignments
	} else {
		assignmentsMap := make(map[wanikaniapi.WKID]*wanikaniapi.Assignment)

		for len(assignmentsMap) < numTargets {
			assignment := assignments[rand.Intn(len(assignments))]

			_, ok := assignmentsMap[assignment.ID]
			if ok {
				continue
			}

			assignmentsMap[assignment.ID] = assignment
			randomAssignments = append(randomAssignments, assignment)
		}
	}

	/*
		randomSubjectIDs := make([]wanikaniapi.WKID, len(randomAssignments))
		for i, assignment := range randomAssignments {
			randomSubjectIDs[i] = assignment.Data.SubjectID
		}
	*/
	randomSubjectIDs := []wanikaniapi.WKID{45}

	subjectPage, err := client.SubjectList(&wanikaniapi.SubjectListParams{
		IDs: randomSubjectIDs,
	})
	if err != nil {
		panic(err)
	}

	logger.Infof("Page = %+v", subjectPage)

	subjectCharacters := make([]string, len(subjectPage.Data))
	for i, subject := range subjectPage.Data {
		switch subject.ObjectType {
		case wanikaniapi.ObjectTypeKanji:
			subjectCharacters[i] = subject.KanjiData.Characters
		case wanikaniapi.ObjectTypeRadical:
			if subject.RadicalData.Characters != nil {
				subjectCharacters[i] = *subject.RadicalData.Characters
			} else {
				subjectCharacters[i] = "(" + subject.RadicalData.Slug + ")"
			}
		case wanikaniapi.ObjectTypeVocabulary:
			subjectCharacters[i] = subject.VocabularyData.Characters
		}
	}

	logger.Infof("Subjects: %s", strings.Join(subjectCharacters, ", "))

	assignmentPage, err := client.AssignmentList(&wanikaniapi.AssignmentListParams{
		SubjectIDs: randomSubjectIDs,
	})
	if err != nil {
		panic(err)
	}

	logger.Infof("Assignment page = %+v", assignmentPage)

	for i, assignment := range assignmentPage.Data {
		logger.Infof("Resurrecting: %v\n", subjectCharacters[i])
		_, err := client.AssignmentResurrect(&wanikaniapi.AssignmentResurrectParams{ID: &assignment.ID})
		if err != nil {
			panic(err)
		}
	}
}

func abort(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}
