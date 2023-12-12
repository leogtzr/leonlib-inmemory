#!/bin/bash

set -e

readonly option="${1}"
readonly webapp_container_name="leonlib_app"
readonly error_wrong_option=81

clean_containers() {
    echo "Removing app containers"

    local -r webapp_container_info=$(docker container ls -a | grep --fixed-strings "${webapp_container_name}")
    if [[ -z "${webapp_container_info}" ]]; then
        return
    fi

    local -r webapp_container_id=$(echo "${webapp_container_info}" | awk '{print $1}')
    if [[ ! -z "${webapp_container_id}" ]]; then
        docker container stop "${webapp_container_id}"
        docker container rm "${webapp_container_id}"
    fi

    echo "Removing app containers - DONE"
}

clean_docker_images() {
    echo "Removing docker images"
    docker images -a | grep --fixed-strings "${webapp_container_name}" | while read docker_image_line; do
        local -r docker_image_id_to_remove=$(echo "${docker_image_line}" | awk '{print $3}')
        if [[ ! -z "${docker_image_id_to_remove}" ]]; then
            docker rmi -f "${docker_image_id_to_remove}"
        fi
    done

    echo "Removing docker images - DONE"
}

remove_unused_images() {
    echo "Removing unused images"
    echo y | docker image prune
    echo "Removing unused images - DONE"
}

case "${option}" in
    cls)
        clean_containers
        clean_docker_images
        remove_unused_images

        ;;

    *)
        echo "Unknown option: ${option}" 2>&1

        exit ${error_wrong_option}
        ;;
esac

exit 0