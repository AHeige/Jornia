package companion

import "context"

// Expiring external world evidence is not permanent personal journey data.
// Remove its cached answers/receipts on startup and before history is read/exported.
func (s *Store) pruneExpiredAnswers(ctx context.Context) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM messages WHERE EXISTS(SELECT 1 FROM json_each(messages.answer,'$.sources') src WHERE julianday(json_extract(src.value,'$.expires_at'))<=julianday('now'))`,
		`DELETE FROM operations WHERE kind='ask' AND status='completed' AND EXISTS(SELECT 1 FROM json_each(operations.result,'$.sources') src WHERE julianday(json_extract(src.value,'$.expires_at'))<=julianday('now'))`,
	} {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return tx.Commit()
}
