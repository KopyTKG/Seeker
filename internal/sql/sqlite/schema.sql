PRAGMA foreign_keys = ON;

BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "Search" (
    "Vendor" TEXT NOT NULL,
    "Product" TEXT,
    PRIMARY KEY ("Vendor")
);


CREATE TABLE IF NOT EXISTS "CVESearch" (
    "Vendor" TEXT NOT NULL,
    "CVE_Id" TEXT NOT NULL,
    FOREIGN KEY("CVE_Id") REFERENCES "CVEs"("Id"),
    FOREIGN KEY("Vendor") REFERENCES "Search"("Vendor")
);


CREATE TABLE IF NOT EXISTS "TermSearch" (
    "Term_Id" INT NOT NULL,
    "Vendor" TEXT NOT NULL,
    FOREIGN KEY("Vendor") REFERENCES "Search"("Vendor"),
    FOREIGN KEY("Term_Id") REFERENCES "Terms"("Id")
);


CREATE TABLE IF NOT EXISTS "CVEs" (
    "Id" TEXT NOT NULL,
    "Title" TEXT NOT NULL,
    "Description" TEXT NOT NULL,
    "CreatedAt" DATETIME NOT NULL,
    "UpdatedAt" DATETIME NOT NULL,
    "Score" FLOAT,
    "Vector" TEXT,
    "Link" TEXT NOT NULL,
    PRIMARY KEY ("Id")
);


CREATE TABLE IF NOT EXISTS "UserTerm" (
    "User_Id" INT NOT NULL,
    "Term_Id" INT NOT NULL,
    FOREIGN KEY("Term_Id") REFERENCES "Terms"("Id"),
    FOREIGN KEY("User_Id") REFERENCES "Users"("Id")
);


CREATE TABLE IF NOT EXISTS "Terms" (
    "Id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "Key" TEXT NOT NULL UNIQUE
);


CREATE TABLE IF NOT EXISTS "Users" (
    "Id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "Tag" TEXT NOT NULL UNIQUE
);

COMMIT;
