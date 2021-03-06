package crawler

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/tosone/GithubTraveler/common/downloader"
	"github.com/tosone/GithubTraveler/common/resp"
	"github.com/tosone/GithubTraveler/models"
	"github.com/tosone/logging"
	uuid "gopkg.in/satori/go.uuid.v1"
)

// repoStargazers get a repo's star num
func repoStargazers(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	var body string
	var err error
	var num uint
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		num++
		var repo = new(models.Repo)
		if repo, err = new(models.Repo).FindByID(num); err != nil {
			if err == gorm.ErrRecordNotFound && num == 1 {
				time.Sleep(time.Second * 30)
			}
			num = 0
			continue
		}
		var stargazersVersion = uuid.NewV4()
		repo.Stargazers = stargazersVersion.String()

		if repo.UserID == 0 {
			time.Sleep(time.Second * 10)
			continue
		}

		var user = new(models.User)
		if user, err = new(models.User).FindByUserID(repo.UserID); err != nil {
			continue
		}
		var nextNum = 1
		for nextNum != 0 {
			if body, nextNum, err = downloader.Get(nextNum, user.Login, repo.Name); err != nil {
				logging.Error(err)
				continue
			}

			var owners []resp.Owner
			if err = json.Unmarshal([]byte(body), &owners); err != nil {
				logging.Error(body)
				logging.Error(err)
				continue
			} else {
				for _, owner := range owners {
					var u = new(models.User)
					u.UserID = owner.ID
					u.Login = owner.Login
					u.Type = owner.Type
					if err = u.Create(); err != nil {
						logging.Error(err)
						continue
					}

					var repoStargazers = new(models.RepoStargazers)
					repoStargazers.UserID = owner.ID
					repoStargazers.RepoID = repo.RepoID
					repoStargazers.Version = stargazersVersion.String()
					if err = repoStargazers.Create(); err != nil {
						logging.Error(err)
						continue
					}
					select {
					case <-ctx.Done():
						return
					default:
					}
				}
			}
			if err = repo.Update(); err != nil {
				logging.Error(err)
			}
		}
	}
}
