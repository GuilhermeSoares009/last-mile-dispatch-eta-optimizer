package dispatch

import "errors"

const minAvailability = 0.5

var ErrNoCouriers = errors.New("no couriers provided")

type Courier struct {
    ID           string
    EtaMinutes   int64
    Availability float64
    Load         float64
}

type Decision struct {
    Courier  Courier
    Score    float64
    Fallback bool
    Reason   string
}

func SelectCourier(couriers []Courier) (Decision, error) {
    if len(couriers) == 0 {
        return Decision{}, ErrNoCouriers
    }

    eligible := make([]Courier, 0, len(couriers))
    for _, courier := range couriers {
        if courier.Availability >= minAvailability {
            eligible = append(eligible, courier)
        }
    }

    if len(eligible) == 0 {
        fallback := pickBest(couriers, nil)
        return Decision{
            Courier:  fallback,
            Score:    ScoreCourier(fallback),
            Fallback: true,
            Reason:   "fallback-no-available-couriers",
        }, nil
    }

    best := pickBest(eligible, nil)
    return Decision{
        Courier:  best,
        Score:    ScoreCourier(best),
        Fallback: false,
        Reason:   "best-score",
    }, nil
}

func SelectWithScores(couriers []Courier, overrides map[string]float64) (Decision, error) {
    if len(couriers) == 0 {
        return Decision{}, ErrNoCouriers
    }

    eligible := make([]Courier, 0, len(couriers))
    for _, courier := range couriers {
        if courier.Availability >= minAvailability {
            eligible = append(eligible, courier)
        }
    }

    if len(eligible) == 0 {
        fallback := pickBest(couriers, overrides)
        return Decision{
            Courier:  fallback,
            Score:    scoreWithOverride(fallback, overrides),
            Fallback: true,
            Reason:   "fallback-no-available-couriers",
        }, nil
    }

    best := pickBest(eligible, overrides)
    return Decision{
        Courier:  best,
        Score:    scoreWithOverride(best, overrides),
        Fallback: false,
        Reason:   "best-score",
    }, nil
}

func pickBest(couriers []Courier, overrides map[string]float64) Courier {
    best := couriers[0]
    bestScore := scoreWithOverride(best, overrides)
    for _, candidate := range couriers[1:] {
        candidateScore := scoreWithOverride(candidate, overrides)
        if candidateScore < bestScore {
            best = candidate
            bestScore = candidateScore
            continue
        }
        if candidateScore == bestScore && candidate.Availability > best.Availability {
            best = candidate
            bestScore = candidateScore
            continue
        }
        if candidateScore == bestScore && candidate.Availability == best.Availability && candidate.ID < best.ID {
            best = candidate
            bestScore = candidateScore
        }
    }
    return best
}

func ScoreCourier(courier Courier) float64 {
    availabilityPenalty := (1 - courier.Availability) * 10
    loadPenalty := courier.Load * 5
    return float64(courier.EtaMinutes) + availabilityPenalty + loadPenalty
}

func scoreWithOverride(courier Courier, overrides map[string]float64) float64 {
    if overrides == nil {
        return ScoreCourier(courier)
    }
    if override, ok := overrides[courier.ID]; ok {
        return override
    }
    return ScoreCourier(courier)
}
