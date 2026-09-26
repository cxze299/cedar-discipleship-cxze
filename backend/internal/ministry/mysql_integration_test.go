//go:build integration

package ministry

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"agp/backend/internal/testdb"
)

func TestEnsureCatalogIsSafeUnderConcurrentFirstLoad(t *testing.T) {
	db := testdb.Open(t)
	testdb.Apply(t, db, "003_ministry_groups.sql")
	testdb.Apply(t, db, "012_ministry_catalog_seed_policy.sql")
	testdb.Exec(t, db, `INSERT INTO study_groups
		(id,code,name,auto_seed_ministry_catalog,created_at,updated_at)
		VALUES (1,'concurrent','Concurrent',1,NOW(),NOW())`)

	repo := NewMySQLRepository(db)
	service := NewService(repo)
	for round := 0; round < 10; round++ {
		testdb.Exec(t, db, `DELETE FROM ministry_groups WHERE study_group_id=1`)

		start := make(chan struct{})
		errs := make(chan error, 8)
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				groups, err := service.Groups(ctx, 1, Actor{})
				if err == nil && len(groups) != len(defaultCatalog) {
					err = fmt.Errorf("group count = %d, want %d", len(groups), len(defaultCatalog))
				}
				errs <- err
			}()
		}
		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("round %d: concurrent catalog initialization failed: %v", round, err)
			}
		}
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ministry_groups WHERE study_group_id=1`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != len(defaultCatalog) {
			t.Fatalf("round %d: catalog count = %d, want %d", round, count, len(defaultCatalog))
		}
	}
}
