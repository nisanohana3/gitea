#!/bin/bash

if [ ! -f custom/conf/app.ini ]
then
    mkdir -p custom/conf/
    echo -e "[server]\nROOT_URL=$(gp url 3000)/" > custom/conf/app.ini
    echo -e "\n[database]\nDB_TYPE = sqlite3\nPATH = $GITPOD_REPO_ROOT/data/gitea.db" >> custom/conf/app.ini
fi
export TAGS="sqlite sqlite_unlock_notify"

make watch
