package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/flow-hydraulics/flow-wallet-api/jobs"
	"github.com/flow-hydraulics/flow-wallet-api/tests/internal/test"
)

func Test_Add_New_Non_Custodial_Account(t *testing.T) {
	cfg := test.LoadConfig(t, testConfigPath)
	svc := test.GetServices(t, cfg).GetAccounts()

	addr := "0x0123456789"

	a, err := svc.AddNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}

	if a.Address != addr {
		t.Fatalf("expected a.Address = %q, got %q", addr, a.Address)
	}
}

func Test_Add_Existing_Non_Custodial_Account_fails(t *testing.T) {
	cfg := test.LoadConfig(t, testConfigPath)
	svc := test.GetServices(t, cfg).GetAccounts()

	addr := "0x0123456789"

	_, err := svc.AddNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.AddNonCustodialAccount(context.Background(), addr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Add_Non_Custodial_Account_After_Delete(t *testing.T) {
	cfg := test.LoadConfig(t, testConfigPath)
	svc := test.GetServices(t, cfg).GetAccounts()

	addr := "0x0123456789"

	_, err := svc.AddNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.DeleteNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}

	// One must be able to add the same account again after it was deleted.
	_, err = svc.AddNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Delete_Non_Existing_Account(t *testing.T) {
	cfg := test.LoadConfig(t, testConfigPath)
	svc := test.GetServices(t, cfg).GetAccounts()

	addr := "0x0123456789"

	err := svc.DeleteNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Delete_Fails_On_Custodial_Account(t *testing.T) {
	cfg := test.LoadConfig(t, testConfigPath)
	svc := test.GetServices(t, cfg).GetAccounts()

	_, a, err := svc.Create(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.DeleteNonCustodialAccount(context.Background(), a.Address)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Delete_Non_Custodial_Account_Is_Idempotent(t *testing.T) {
	cfg := test.LoadConfig(t, testConfigPath)
	svc := test.GetServices(t, cfg).GetAccounts()

	addr := "0x0123456789"

	_, err := svc.AddNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.DeleteNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.DeleteNonCustodialAccount(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
}

// Test if the service is able to concurrently create multiple accounts
func Test_Add_Multiple_New_Custodial_Accounts(t *testing.T) {
	cfg := test.LoadConfig(t, testConfigPath)

	// TODO: this test fails if instanceCount > 1 (database is locked, with sqlite)
	instanceCount := 1
	accountsToCreate := instanceCount * 5
	// Worst case scenario where theoretically maximum number of transactions are done concurrently
	// TODO: database is locked (with sqlite)
	// cfg.WorkerCount = uint(accountsToCreate / instanceCount)

	svcs := make([]test.Services, instanceCount)

	for i := 0; i < instanceCount; i++ {
		svcs[i] = test.GetServices(t, cfg)
	}

	if cfg.AdminProposalKeyCount <= 1 {
		t.Skip("skipped as \"cfg.AdminProposalKeyCount\" is less than or equal to 1")
	}

	if accounts, err := svcs[0].GetAccounts().List(0, 0); err != nil {
		t.Fatal(err)
	} else if len(accounts) > 1 {
		t.Fatal("expected there to be only 1 account")
	}

	wg := sync.WaitGroup{}
	errChan := make(chan error, accountsToCreate*4)

	for i := 0; i < accountsToCreate; i++ {
		wg.Add(1)
		go func(i int, svcs []test.Services) {
			defer wg.Done()

			svc := svcs[i%instanceCount].GetAccounts()
			jobSvc := svcs[i%instanceCount].GetJobs()

			job, _, err := svc.Create(context.Background(), false)
			if err != nil {
				errChan <- err
				return
			}

			for job.State == jobs.Init || job.State == jobs.Accepted || job.State == jobs.Error {
				time.Sleep(100 * time.Millisecond)
				if j, err := jobSvc.Details(job.ID.String()); err != nil {
					continue
				} else {
					job = j
				}
			}

			if job.State == jobs.Failed {
				errChan <- fmt.Errorf(job.Error)
				return
			}
		}(i, svcs)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		t.Fatal(err)
	default:
	}

	if accounts, err := svcs[0].GetAccounts().List(0, 0); err != nil {
		t.Fatal(err)
	} else if len(accounts) < 1+accountsToCreate {
		t.Fatalf("expected there to be %d accounts", 1+accountsToCreate)
	}
}
