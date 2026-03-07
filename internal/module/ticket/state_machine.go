package ticket

import "fmt"

var allowedStatusTransitions = map[TicketStatus]map[TicketStatus]struct{}{
	TicketStatusOpen: {
		TicketStatusInProgress: {},
		TicketStatusClosed:     {},
	},
	TicketStatusInProgress: {
		TicketStatusResolved: {},
		TicketStatusClosed:   {},
	},
	TicketStatusResolved: {
		TicketStatusClosed: {},
	},
	TicketStatusClosed: {},
}

func ValidateStatusTransition(from, to TicketStatus) error {
	if from == to {
		return nil
	}

	nextStatuses, ok := allowedStatusTransitions[from]
	if !ok {
		return fmt.Errorf("unknown current status: %s", from)
	}

	// 当前阶段采用显式状态机，而不是直接写字符串更新，目的是把主流程规则集中管理，方便后续 #4 审计与 #8 AI 动作复用。
	if _, ok := nextStatuses[to]; !ok {
		return fmt.Errorf("transition from %s to %s is not allowed", from, to)
	}

	return nil
}
