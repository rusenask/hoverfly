package matching

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/SpectoLabs/hoverfly/core/cache"
	"github.com/SpectoLabs/hoverfly/core/handlers/v2"
	"github.com/SpectoLabs/hoverfly/core/models"
)

type CacheMatcher struct {
	RequestCache cache.Cache
	Webserver    *bool
}

// getResponse returns stored response from cache
func (this *CacheMatcher) GetResponse(req *models.RequestDetails) (*models.ResponseDetails, *MatchingError) {
	if this.RequestCache == nil {
		return nil, &MatchingError{
			Description: "No cache set",
		}
	}

	log.Debug("Checking cache for request")

	var key string

	if *this.Webserver {
		key = req.HashWithoutHost()
	} else {
		key = req.Hash()
	}

	pairBytes, err := this.RequestCache.Get([]byte(key))

	if err != nil {
		log.WithFields(log.Fields{
			"key":         key,
			"error":       err.Error(),
			"query":       req.Query,
			"path":        req.Path,
			"destination": req.Destination,
			"method":      req.Method,
		}).Debug("Failed to retrieve response from cache")

		return nil, &MatchingError{
			StatusCode:  412,
			Description: "Could not find recorded request in cache",
		}
	}

	// getting cache response
	pair, err := models.NewRequestResponsePairFromBytes(pairBytes)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"value": string(pairBytes),
			"key":   key,
		}).Debug("Failed to decode payload from cache")
		return nil, &MatchingError{
			StatusCode:  500,
			Description: "Failed to decode payload from cache",
		}
	}

	log.WithFields(log.Fields{
		"key":         key,
		"path":        req.Path,
		"rawQuery":    req.Query,
		"method":      req.Method,
		"destination": req.Destination,
		"status":      pair.Response.Status,
	}).Info("Response found interface{} cache")

	return &pair.Response, nil
}

func (this CacheMatcher) GetAllResponses() ([]v2.RequestResponsePairViewV1, error) {
	if this.RequestCache == nil {
		return nil, &MatchingError{
			Description: "No cache set",
		}
	}

	records, err := this.RequestCache.GetAllEntries()
	if err != nil {
		return []v2.RequestResponsePairViewV1{}, err
	}

	pairViews := []v2.RequestResponsePairViewV1{}

	for _, v := range records {
		if pair, err := models.NewRequestResponsePairFromBytes(v); err == nil {
			pairView := pair.ConvertToRequestResponsePairView()
			pairViews = append(pairViews, pairView)
		} else {
			log.Error(err)
			return []v2.RequestResponsePairViewV1{}, err
		}
	}

	return pairViews, nil
}

func (this *CacheMatcher) SaveRequestResponsePair(pair *models.RequestResponsePair) error {
	if this.RequestCache == nil {
		return errors.New("No cache set")
	}

	var key string

	if *this.Webserver {
		key = pair.IdWithoutHost()
	} else {
		key = pair.Id()
	}

	log.WithFields(log.Fields{
		"path":          pair.Request.Path,
		"rawQuery":      pair.Request.Query,
		"requestMethod": pair.Request.Method,
		"bodyLen":       len(pair.Request.Body),
		"destination":   pair.Request.Destination,
		"hashKey":       key,
	}).Debug("Saving response to cache")

	pairBytes, err := pair.Encode()

	if err != nil {
		return err
	}

	return this.RequestCache.Set([]byte(key), pairBytes)
}

func (this CacheMatcher) FlushCache() error {
	if this.RequestCache == nil {
		return errors.New("No cache set")
	}

	return this.RequestCache.DeleteData()
}