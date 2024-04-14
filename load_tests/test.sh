#!/bin/bash
# Запуск сервера
gnome-terminal -- bash -c "make run; exec bash" &

# Даем серверу некоторое время для запуска
sleep 5 

gnome-terminal -- bash -c "vegeta attack -duration=60s -rate=1000 -targets=./load_tests/targets.txt | vegeta report > ./load_tests/vegeta_test.out; exec bash"
