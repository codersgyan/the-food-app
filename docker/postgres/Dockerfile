# Postgres 18 with PostGIS, for the spatial index on restaurants (migration
# 00011). The official postgis/postgis image is built only for x86, and our
# server is ARM, so we add PostGIS to the official Postgres image instead.
FROM postgres:18

RUN apt-get update \
 && apt-get install -y --no-install-recommends postgresql-18-postgis-3 \
 && rm -rf /var/lib/apt/lists/*
