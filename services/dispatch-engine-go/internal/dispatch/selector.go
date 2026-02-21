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
        fallback := pickBest(couriers)
        return Decision{
            Courier:  fallback,
            Score:    scoreCourier(fallback),
            Fallback: true,
            Reason:   "fallback-no-available-couriers",
        }, nil
    }

    best := pickBest(eligible)
    return Decision{
        Courier:  best,
        Score:    scoreCourier(best),
        Fallback: false,
        Reason:   "best-score",
    }, nil
}

func pickBest(couriers []Courier) Courier {
    best := couriers[0]
    bestScore := scoreCourier(best)
    for _, candidate := range couriers[1:] {
        candidateScore := scoreCourier(candidate)
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

func scoreCourier(courier Courier) float64 {
    availabilityPenalty := (1 - courier.Availability) * 10
    loadPenalty := courier.Load * 5
    return float64(courier.EtaMinutes) + availabilityPenalty + loadPenalty
}
