#!/bin/bash

if [ ! -f custom/conf/app.ini ]
then
    mkdir -p custom/conf/
    echo -e "\n[database]\nDB_TYPE = sqlite3\nPATH = ${containerWorkspaceFolderBasename}/data/gitea.db" >> custom/conf/app.ini
fi
