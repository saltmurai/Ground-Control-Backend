CREATE TABLE "packages"(
    "id" bigserial NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "weight" DOUBLE PRECISION NOT NULL,
    "height" DOUBLE PRECISION NOT NULL,
    "length" DOUBLE PRECISION NOT NULL,
    "sender_id" UUID NOT NULL,
    "receiver_id" UUID NOT NULL
);
ALTER TABLE
    "packages" ADD PRIMARY KEY("id");
CREATE TABLE "sequences"(
    "id" bigserial NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "description" TEXT NOT NULL,
    "seq" jsonb NOT NULL,
    "length" BIGINT NOT NULL
);
ALTER TABLE
    "sequences" ADD PRIMARY KEY("id");
CREATE TABLE "users"(
    "id" UUID NOT NULL,
    "name" TEXT NOT NULL
);
ALTER TABLE
    "users" ADD PRIMARY KEY("id");
CREATE TABLE "missions"(
    "id" bigserial NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "drone_id" BIGINT NOT NULL,
    "package_id" BIGINT NOT NULL,
    "seq_id" BIGINT NOT NULL,
    "image_folder" VARCHAR(255) NOT NULL,
    "status" VARCHAR(255) NOT NULL,
    "path" jsonb NOT NULL
);
ALTER TABLE
    "missions" ADD PRIMARY KEY("id");
CREATE TABLE "drones"(
    "id" bigserial NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "address" VARCHAR(255) NOT NULL,
    "ip" VARCHAR(255) NOT NULL,
    "status" BOOLEAN NOT NULL,
    "port" BIGINT NOT NULL
);
ALTER TABLE
    "drones" ADD PRIMARY KEY("id");
ALTER TABLE
    "packages" ADD CONSTRAINT "packages_receiver_id_foreign" FOREIGN KEY("receiver_id") REFERENCES "users"("id");
ALTER TABLE
    "packages" ADD CONSTRAINT "packages_sender_id_foreign" FOREIGN KEY("sender_id") REFERENCES "users"("id");
ALTER TABLE
    "missions" ADD CONSTRAINT "missions_package_id_foreign" FOREIGN KEY("package_id") REFERENCES "packages"("id");
ALTER TABLE
    "missions" ADD CONSTRAINT "missions_drone_id_foreign" FOREIGN KEY("drone_id") REFERENCES "drones"("id");
ALTER TABLE
    "missions" ADD CONSTRAINT "missions_seq_id_foreign" FOREIGN KEY("seq_id") REFERENCES "sequences"("id");