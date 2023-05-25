CREATE TABLE "packages"(
    "id" BIGINT NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "weight" DOUBLE PRECISION NOT NULL
);
ALTER TABLE
    "packages" ADD PRIMARY KEY("id");
CREATE TABLE "sequences"(
    "id" bigserial NOT NULL,
    "name" VARCHAR(255) NULL,
    "description" TEXT NULL,
    "seq" jsonb NULL,
    "created_at" DATE NOT NULL
);
ALTER TABLE
    "sequences" ADD PRIMARY KEY("id");
CREATE TABLE "missions"(
    "id" BIGINT NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "drone_id" BIGINT NOT NULL,
    "package_id" BIGINT NOT NULL,
    "seq_id" BIGINT NOT NULL
);
ALTER TABLE
    "missions" ADD PRIMARY KEY("id");
CREATE TABLE "drones"(
    "id" BIGINT NOT NULL,
    "ip" VARCHAR(255) NOT NULL,
    "name" VARCHAR(255) NOT NULL
);
ALTER TABLE
    "drones" ADD PRIMARY KEY("id");
ALTER TABLE
    "missions" ADD CONSTRAINT "missions_package_id_foreign" FOREIGN KEY("package_id") REFERENCES "packages"("id");
ALTER TABLE
    "missions" ADD CONSTRAINT "missions_drone_id_foreign" FOREIGN KEY("drone_id") REFERENCES "drones"("id");
ALTER TABLE
    "missions" ADD CONSTRAINT "missions_seq_id_foreign" FOREIGN KEY("seq_id") REFERENCES "sequences"("id");