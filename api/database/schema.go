package database

import (
	"database/sql"
	"fmt"
)

func CreateSchema(db *sql.DB) error {
	// better-auth tables (same as web/better-auth_migrations), so a fresh DB works without `npm run migrate`
	authUser := `CREATE TABLE IF NOT EXISTS "user" ("id" text not null primary key, "name" text not null, "email" text not null unique, "emailVerified" boolean not null, "image" text, "createdAt" timestamp not null, "updatedAt" timestamp not null, "username" text unique, "displayUsername" text);`
	// columns added by better-auth's username plugin, for DBs created before it was enabled
	authUsernameCol := `ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "username" text UNIQUE;`
	authDisplayUsernameCol := `ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "displayUsername" text;`
	authSession := `CREATE TABLE IF NOT EXISTS "session" ("id" text not null primary key, "expiresAt" timestamp not null, "token" text not null unique, "createdAt" timestamp not null, "updatedAt" timestamp not null, "ipAddress" text, "userAgent" text, "userId" text not null references "user" ("id"));`
	authAccount := `CREATE TABLE IF NOT EXISTS "account" ("id" text not null primary key, "accountId" text not null, "providerId" text not null, "userId" text not null references "user" ("id"), "accessToken" text, "refreshToken" text, "idToken" text, "accessTokenExpiresAt" timestamp, "refreshTokenExpiresAt" timestamp, "scope" text, "password" text, "createdAt" timestamp not null, "updatedAt" timestamp not null);`
	authVerification := `CREATE TABLE IF NOT EXISTS "verification" ("id" text not null primary key, "identifier" text not null, "value" text not null, "expiresAt" timestamp not null, "createdAt" timestamp, "updatedAt" timestamp);`

	enumCheck := `DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'challenge_state') THEN
        CREATE TYPE challenge_state AS ENUM ('pending', 'active', 'denied', 'complete');
    END IF;
END$$;`

	challenges := `CREATE TABLE IF NOT EXISTS "challenges" (
    "id" TEXT NOT NULL PRIMARY KEY, 
		"challenger" TEXT NOT NULL REFERENCES "user" ("id"),
		"challengee" TEXT NOT NULL REFERENCES "user" ("id"),
    "state" challenge_state NOT NULL, 
    "createdBy" TEXT NOT NULL REFERENCES "user" ("id"),
    "createdAt" TIMESTAMP NOT NULL
);`

	games := `CREATE TABLE IF NOT EXISTS "games" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "challengeId" TEXT NOT NULL REFERENCES "challenges" ("id"),
    "state" TEXT NOT NULL,
    "createdAt" TIMESTAMP NOT NULL
);`

	// match format and result columns, for DBs created before best-of-N sets
	challengesBestOf := `ALTER TABLE "challenges" ADD COLUMN IF NOT EXISTS "bestOf" integer NOT NULL DEFAULT 1;`
	gamesWinner := `ALTER TABLE "games" ADD COLUMN IF NOT EXISTS "winnerId" text REFERENCES "user" ("id");`
	gamesScore := `ALTER TABLE "games" ADD COLUMN IF NOT EXISTS "score" text;`
	gamesProposedBy := `ALTER TABLE "games" ADD COLUMN IF NOT EXISTS "proposedBy" text REFERENCES "user" ("id");`

	userStats := `CREATE TABLE IF NOT EXISTS "user_stats"(
		"userId" TEXT NOT NULL PRIMARY KEY REFERENCES "user" ("id"),
		"mmr" INTEGER NOT NULL DEFAULT 1500,
		"wins" INTEGER NOT NULL DEFAULT 0,
		"losses" INTEGER NOT NULL DEFAULT 0,
		"updatedAt" TIMESTAMP NOT NULL DEFAULT NOW()
	)`

	for _, query := range []string{authUser, authUsernameCol, authDisplayUsernameCol, authSession, authAccount, authVerification, enumCheck, challenges, games, userStats, challengesBestOf, gamesWinner, gamesScore, gamesProposedBy} {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("error creating schema: %w", err)
		}
	}
	return nil
}
