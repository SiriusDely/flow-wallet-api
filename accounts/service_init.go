package accounts

import (
	"context"
	"errors"
	"strings"

	"github.com/flow-hydraulics/flow-wallet-api/keys"
	"github.com/flow-hydraulics/flow-wallet-api/templates/template_strings"
	"github.com/flow-hydraulics/flow-wallet-api/transactions"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	log "github.com/sirupsen/logrus"
)

func (s *Service) InitAdminAccount(ctx context.Context) error {
	log.Debug("Initializing admin account")

	a, err := s.store.Account(s.cfg.AdminAddress)
	if err != nil {
		if !strings.Contains(err.Error(), "record not found") {
			return err
		}
		// Admin account not in database
		a = Account{Address: s.cfg.AdminAddress}
		err := s.store.InsertAccount(&a)
		if err != nil {
			return err
		}
		AccountAdded.Trigger(AccountAddedPayload{
			Address: flow.HexToAddress(s.cfg.AdminAddress),
		})
	}

	log.WithFields(log.Fields{
		"wantedCount": s.cfg.AdminProposalKeyCount,
	}).Debug("Checking admin account proposal keys")

	if err := s.km.CheckAdminProposalKeyCount(ctx); err != nil {
		if errors.Is(err, keys.ErrAdminProposalKeyCountMismatch) {
			log.WithFields(log.Fields{
				"err": err,
			}).Info("Admin proposal key count mismatch")

			keyCount, err := s.km.InitAdminProposalKeys(ctx)
			if err != nil {
				return err
			}

			if keyCount < s.cfg.AdminProposalKeyCount {
				err = s.addAdminProposalKeys(ctx, s.cfg.AdminProposalKeyCount-keyCount)
				if err != nil {
					return err
				}

				_, err = s.km.InitAdminProposalKeys(ctx)
				if err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}

	return nil
}

func (s *Service) addAdminProposalKeys(ctx context.Context, count uint16) error {

	log.
		WithFields(log.Fields{"count": count}).
		Info("Adding admin account proposal keys")

	code := template_strings.AddProposalKeyTransaction
	args := []transactions.Argument{
		cadence.NewInt(s.cfg.AdminKeyIndex),
		cadence.NewUInt16(count),
	}

	_, _, err := s.transactions.Create(ctx, true, s.cfg.AdminAddress, code, args, transactions.General)
	return err
}
