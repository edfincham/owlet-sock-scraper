package models

import (
	"context"
	"fmt"
	"time"

	db "github.com/edfincham/owlet-sock-scraper/db"
	sock "github.com/edfincham/owlet-sock-scraper/sock"

	"github.com/jackc/pgx/v5"
)

func InsertVitals(db *db.PostgresPool, v sock.Vitals, serial string) error {
	query := `
		INSERT INTO Vitals (
			ts, ox, hr, mv, sc, st, bso, bat, btt, chg, aps, alrt, ota, srf, rsi,
			sb, ss, mvb, oxta, onm, bsb, mrs, hw, serial
		) VALUES (
			@ts, @ox, @hr, @mv, @sc, @st, @bso, @bat, @btt, @chg, @aps, @alrt, @ota, @srf, @rsi,
			@sb, @ss, @mvb, @oxta, @onm, @bsb, @mrs, @hw, @serial
		)`

	now := time.Now()
	timestamp := time.Unix(now.Unix(), 0)

	args := pgx.NamedArgs{
		"ts":     timestamp,
		"ox":     v.Ox,
		"hr":     v.Hr,
		"mv":     v.Mv,
		"sc":     v.Sc,
		"st":     v.St,
		"bso":    v.Bso,
		"bat":    v.Bat,
		"btt":    v.Btt,
		"chg":    v.Chg,
		"aps":    v.Aps,
		"alrt":   v.Alrt,
		"ota":    v.Ota,
		"srf":    v.Srf,
		"rsi":    v.Rsi,
		"sb":     v.Sb,
		"ss":     v.Ss,
		"mvb":    v.Mvb,
		"oxta":   v.Oxta,
		"onm":    v.Onm,
		"bsb":    v.Bsb,
		"mrs":    v.Mrs,
		"hw":     v.Hw,
		"serial": serial,
	}

	_, err := db.DB.Exec(context.Background(), query, args)
	if err != nil {
		return fmt.Errorf("failed to insert vitals: %w", err)
	}

	return nil
}
