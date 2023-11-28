package scanner

import (
	"github.com/seanime-app/seanime/internal/anilist"
	"github.com/seanime-app/seanime/internal/anizip"
	"github.com/seanime-app/seanime/internal/entities"
	"github.com/seanime-app/seanime/internal/limiter"
	"github.com/seanime-app/seanime/internal/util"
	"testing"
)

func TestScanner_Scan(t *testing.T) {

	baseMediaCache := anilist.NewBaseMediaCache()
	anizipCache := anizip.NewCache()
	anilistRateLimiter := limiter.NewAnilistLimiter()
	anilistClient := anilist.MockGetAnilistClient()
	logger := util.NewLogger()
	media := anilist.MockGetAllMedia()

	// Set base media cache
	for _, m := range *media {
		baseMediaCache.Set(m.ID, m)
	}

	// Get local files
	localFiles, ok := entities.MockGetLocalFiles()
	if !ok {
		t.Fatal("expected local files, got error")
	}

	// Create a new container for media
	mc := NewMediaContainer(&MediaContainerOptions{
		allMedia: *media,
	})

	// Create a new matcher
	matcher := &Matcher{
		localFiles:     localFiles,
		mediaContainer: mc,
		baseMediaCache: baseMediaCache,
		logger:         logger,
	}

	// Match local files with media
	err := matcher.MatchLocalFilesWithMedia()
	if err != nil {
		t.Fatal("expected result, got error:", err.Error())
	}

	// Create a new hydrator
	hydrator := &FileHydrator{
		Media:              mc.allMedia,
		LocalFiles:         localFiles,
		AnizipCache:        anizipCache,
		AnilistClient:      anilistClient,
		BaseMediaCache:     baseMediaCache,
		AnilistRateLimiter: anilistRateLimiter,
		Logger:             logger,
	}
	hydrator.HydrateMetadata()

	for _, lf := range localFiles {
		if lf == nil {
			t.Fatal("expected base media, got nil")
		}
		t.Logf("LocalFile: %+v\nParsed: %+v\nMetadata: %+v\n\n", lf, lf.ParsedData, lf.Metadata)
	}

}
