package crawler

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"github.com/tosone/GithubTraveler/common/downloader"
	"github.com/tosone/GithubTraveler/common/resp"
	"github.com/tosone/GithubTraveler/models"
	"github.com/tosone/logging"
	uuid "gopkg.in/satori/go.uuid.v1"
)

// userFollowers get all of specified user's followers
func userFollowers(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	var body string
	var err error
	var num uint
	for {
		num++
		var user = new(models.User)
		if user, err = new(models.User).FindByID(num); err != nil {
			if err == gorm.ErrRecordNotFound && num == 1 {
				time.Sleep(time.Second * time.Duration(viper.GetInt("Crawler.WaitDataReady")))
			}
			num = 0
			continue
		}
		if user.UserID == 0 {
			time.Sleep(time.Second * time.Duration(viper.GetInt("Crawler.WaitDataReady")))
			continue
		}
		var followersVersion = uuid.NewV4()
		user.Followers = followersVersion.String()

		var nextNum = 1

		for nextNum != 0 {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if body, nextNum, err = downloader.Get(nextNum, user.Login); err != nil {
				logging.Error(err)
				continue
			}

			var owners []resp.Owner
			if err = json.Unmarshal([]byte(body), &owners); err != nil {
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
					var reliableFollowers = new(models.UserFollowers)
					reliableFollowers.UserID = user.UserID
					reliableFollowers.Version = followersVersion.String()
					reliableFollowers.FollowerUserID = u.UserID
					if err = reliableFollowers.Create(); err != nil {
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
		}

		if err = user.Update(); err != nil {
			logging.Error(err)
		}
	}
}
