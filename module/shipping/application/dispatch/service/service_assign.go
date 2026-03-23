package service

import "context"

// AssignMarks delegates mark assignment to AddMarks.
func (s *Service) AssignMarks(ctx context.Context, command AddMarksCommand) error {
	_, err := s.AddMarks(ctx, command)

	return err
}
