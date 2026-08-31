#!/usr/bin/env bash

function usage() {
    uargs=(
        $0
        -f release-file
        -t version-tag
        -h help
    )
    echo -e "usage:\n  ${uargs[@]}\n"
}

function info() {
    echo -e "$(date +%T) $*"
}

function warn() {
    # local color nocolor magenta=$(tput setaf 5) cyan=$(tput setaf 6) reset=$(tput sgr0)
    local color nocolor magenta=$(tput setaf 5) reset=$(tput sgr0)
    echo

    color=$magenta
    nocolor=$reset

    echo -e "${color}WARN: $*${nocolor}"

    return 0
}

function error() {
    # local color nocolor yellow=$(tput setaf 3) reset=$(tput sgr0)
    local color nocolor red=$(tput setaf 1) reset=$(tput sgr0)

    printf "\n" >&2

    color=$red
    nocolor=$reset

    printf "%s\n" "${color}ERROR: $*${nocolor}" >&2

    return 0
}

function publish_docker_image() {

    local OPTIND=1
    # local OPTARG
    local pname pdir cfgdir tag duser dtoken sdesc

    while getopts "n:s:d:c:t:U:P:" opt; do
        case $opt in
            n) pname=$OPTARG;;
            s) sdesc=$OPTARG;;
            d) pdir=$OPTARG;;
            c) cfgdir=$OPTARG;;
            t) tag=$OPTARG;;
            U) duser=$OPTARG;;
            P) dtoken="$OPTARG";;
            *) error "publish_docker_image(): unknown option: $opt";
               return 1;;
        esac
    done
    shift $((OPTIND -1))

    # info "Publishing images ($duser/$pname:latest, $duser/$pname:$tag) to docker hub."
    info "Publishing image '$duser/$pname:latest' to docker hub."

    cd $pdir

    info "Generating '$cfgdir/config.json' using docker hub token."

    info "Pruning existing images."

    docker image rm $duser/$pname:latest >/dev/null 2>&1
    docker container prune -f >/dev/null 2>&1
    docker buildx prune -a -f >/dev/null 2>&1
    rm -rf ~/.docker/buildx

    info "Building image."

    docker build -q --no-cache -t $duser/$pname:latest . >/dev/null 2>&1

    info "Pushing image."

    export DOCKER_CONFIG="$cfgdir"

    rm -f $DOCKER_CONFIG/config.json

    docker login --password-stdin -u $duser >/dev/null 2>/dev/null <<<"$dtoken" || {
        local rc=$?
        error "publish_docker_image(): unable to login to docker hub as '$duser' [$rc]."
        return $rc
    }

    ls -d $DOCKER_CONFIG/config.json >/dev/null || return

    docker push $duser/$pname:latest >/dev/null 2>&1 || {
        local rc=$?
        error "publish_docker_image(): unable to publish image '$duser/$pname:latest' [$rc]."
        return $rc
    }

    info "Updating image description."

    local ctypehdr="Content-Type: application/json"
    local drepourl="https://hub.docker.com/v2/repositories/$duser/$pname/"
    local dtokenurl="https://hub.docker.com/v2/users/login/"

    jstr="$(jq -n -c --arg username "$duser" --arg dtoken "$dtoken" '{
                username: $username, password: $dtoken }')"

    cargs=(
        -4
        -sSLkf
        -m 30
        --retry 3
        --retry-delay 2
        -H "$ctypehdr"
        -X POST
    )

    jwtobj=$(curl "${cargs[@]}" -d @- "$dtokenurl" <<<"$jstr") || {
        local rc=$?
        error "publish_docker_image(): unable to get jwt token [$rc]."
        return $rc
    }

    jwt="$(jq -r .token <<<"$jwtobj")"

    [[ "$jwt" ]] || {
        error "publish_docker_image(): got empty jwt token."
        return 1
    }

    # echo "$jwt"

    authhdr="Authorization: JWT $jwt"

    jstr="$(jq -n -c --arg short "$sdesc" --rawfile full DOCKER.md '{
                description: $short, full_description: $full }')"

    cargs=(
        -4
        -sSLkf
        -m 60
        --speed-time 30
        --speed-limit 1000
        -w "%{http_code}\n"
        -H "$authhdr"
        -H "$ctypehdr"
        -X PATCH
    )

    curl "${cargs[@]}" -d @- "$drepourl" -o /dev/null <<<"$jstr" >/dev/null 2>&1 || {
        local rc=$?
        error "publish_docker_image(): unable to update image description [$rc]."
        return $rc
    }

    return 0
}

function publish_github() {

    local OPTIND=1
    # local OPTARG
    local pname datadir tag guser gtoken

    while getopts "n:D:t:u:p:" opt; do
        case $opt in
            n) pname=$OPTARG;;
            D) datadir=$OPTARG;;
            t) tag=$OPTARG;;
            u) guser=$OPTARG;;
            p) gtoken=$OPTARG;;
            *) error "publish_github(): unknown option: $opt";
               return 1;;
        esac
    done
    shift $((OPTIND -1))

    info "Publishing '$pname', tag '$tag' to github."

    cd $datadir || return

    local relfiles=(
        "${pname}-linux-amd64.tar.gz"
        "${pname}-darwin-arm64.tar.gz"
        "${pname}-windows-amd64.zip"
        "checksums.txt"
    )

    export GH_TOKEN="$gtoken"
    export GH_REPO="$guser/$pname"

    gh api "repos/$GH_REPO/git/ref/tags/$tag" &>/dev/null || {
        local rc=$?
        error "publish_github(): Tag '$tag' does not exist on $GH_REPO. Aborting release creation [$rc]."
        return $rc
    }

    gh release view "$tag" &>/dev/null && {
        # gh release delete "$tag" --cleanup-tag -y
        # gh release delete "$tag" -y >/dev/null
        gh release delete "$tag" -y
    }

    gh release create "$tag" "${relfiles[@]}" --title "$tag" --notes "Release $tag" >/dev/null || {
        local rc=$?
        error "publish_github(): unable to create release for $GH_REPO, tag '$tag' [$rc]."
        return $rc
    }

    return 0
}

function build_artifacts() {

    local OPTIND=1
    # local OPTARG
    local pname pdir datadir tag

    while getopts "n:d:D:t:" opt; do
        case $opt in
            n) pname=$OPTARG;;
            d) pdir=$OPTARG;;
            D) datadir=$OPTARG;;
            t) tag=$OPTARG;;
            *) error "build_artifacts(): unknown option: $opt";
               return 1;;
        esac
    done
    shift $((OPTIND -1))

    info "Building artifacts for '$pname', tag '$tag'."

    cd $pdir

    rm -f $pname $datadir/${pname}-linux-amd64.tar.gz

    go mod tidy &&
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $pname . &&
    ls $pname >/dev/null &&
    tar -czf $datadir/${pname}-linux-amd64.tar.gz $pname &&
    ls $datadir/${pname}-linux-amd64.tar.gz >/dev/null || {
        local rc=$?
        error "build_artifacts(): unable to build project $pname for $GOOS.$GOARCH [$rc]."
        return $rc
    }

    rm -f $pname

    rm -f $pname $datadir/${pname}-darwin-arm64.tar.gz

    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $pname . &&
    ls $pname >/dev/null &&
    tar -czf $datadir/${pname}-darwin-arm64.tar.gz $pname &&
    ls $datadir/${pname}-darwin-arm64.tar.gz >/dev/null || {
        local rc=$?
        error "build_artifacts(): unable to build project $pname for $GOOS.$GOARCH [$rc]."
        return $rc
    }

    rm -f $pname

    rm -f ${pname}.exe $datadir/${pname}-windows-amd64.zip

    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ${pname}.exe . &&
    ls ${pname}.exe >/dev/null &&
    zip -q $datadir/${pname}-windows-amd64.zip ${pname}.exe &&
    ls $datadir/${pname}-windows-amd64.zip >/dev/null || {
        local rc=$?
        error "build_artifacts(): unable to build project $pname for $GOOS.$GOARCH [$rc]."
        return $rc
    }

    rm -f ${pname}.exe

    info "Creating checksums for tag '$tag'."

    cd $datadir

    rm -f checksums.txt

    sha256sum ${pname}-*.tar.gz ${pname}-*.zip > checksums.txt &&
    ls checksums.txt >/dev/null || {
        local rc=$?
        error "build_artifacts(): unable to build checksums for $pname [$rc]."
        return $rc
    }

    return 0
}

function main() {

    local OPTIND=1
    # local OPTARG

    local pcfgfile tag

    while getopts "f:t:h" opt; do
        case $opt in
            f) pcfgfile=$OPTARG;;
            t) tag=$OPTARG;;
            h) usage;
               exit 0;;
            *) error "main(): unknown option: $opt";
               usage;
               exit 1;;
        esac
    done
    shift $((OPTIND -1))

    [[ "$pcfgfile" && -f "$pcfgfile" ]] || {
        error "main(): config file [$pcfgfile] is not specified or not found."
        usage
        return 1
    }

    # ls -d "$pcfgfile" >/dev/null || return
    pcfgfile=$(realpath -e "$pcfgfile") || return

    [[ "$tag" ]] || {
        error "main(): version tag must be specified."
        usage
        return 1
    }

    local pname pdir sdesc cfgdir datadir ghost guser gtoken durl duser dtoken

    IFS="|" read -r pname pdir sdesc cfgdir datadir ghost guser gtoken durl duser dtoken < <(yq eval '
        [
            .project-name,
            .project-dir,
            .project-short-desc,
            .config-dir,
            .data-dir,
            .github.host // "github.com",
            .github.user,
            .github.token,
            .docker.url // "https://index.docker.io/v1/",
            .docker.user,
            .docker.token
        ] | map(. // "") | join("|")
    ' "$pcfgfile")

    [[ "$pname" ]] || {
        error "main(): project name not specified."
        return 1
    }

    [[ "$sdesc" ]] || {
        error "main(): project short description not specified."
        return 1
    }

    [[ "$pdir" && -d "$pdir" ]] || {
        error "main(): project directory not specified or not found."
        return 1
    }

    cd $pdir >/dev/null || {
        local rc=$?
        error "main(): project directory '$pdir' does not exist [$rc]."
        return $rc
    }

    XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
    cfgdir="${cfgdir:-$XDG_CONFIG_HOME/$pname}"

    mkdir -p $cfgdir
    cfgdir=$(realpath -e "$cfgdir") || return

    [[ "$cfgdir" && -d "$cfgdir" ]] || {
        error "main(): config directory not specified or not found."
        return 1
    }

    cd $cfgdir >/dev/null || {
        local rc=$?
        error "main(): config directory '$cfgdir' does not exist [$rc]."
        return $rc
    }

    # make sure pcfgfile is not colliding with the docker config file.
    #
    [[ "$pcfgfile" == "$cfgdir/config.json" ]] && {
        error "main(): file specified on command-line cannot be $cfgdir/config.json"
        return 1
    }

    XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
    datadir="${datadir:-$XDG_DATA_HOME/$pname}"

    mkdir -p $datadir
    datadir=$(realpath -e "$datadir") || return

    [[ "$datadir" && -d "$datadir" ]] || {
        error "main(): data directory not specified or not found."
        return 1
    }

    cd $datadir >/dev/null || {
        local rc=$?
        error "main(): data directory '$datadir' does not exist [$rc]."
        return $rc
    }

    # make sure pcfgfile is not colliding with file that would be created under $datadir.
    #
    if [[ "$(dirname "$pcfgfile")" == "$datadir" ]]; then
        # [[ "$pcfgfile" =~ \.(tar\.gz|zip|txt)$ ]]

        case "$pcfgfile" in
            *.tar.gz|*.zip|*.txt)
                error "main(): file specified on command-line cannot be $datadir/{.tar.gz, .zip, .txt}."
                return 1
                ;;
        esac
    fi

    [[ "$guser" ]] || {
        error "main(): github username not specified."
        return 1
    }

    [[ "$gtoken" ]] || {
        error "main(): github token not specified."
        return 1
    }

    [[ "$duser" ]] || {
        error "main(): docker hub username not specified."
        return 1
    }

    [[ "$dtoken" ]] || {
        error "main(): docker hub token not specified."
        return 1
    }

    local bargs=(-n "$pname" -d "$pdir" -D "$datadir" -t "$tag")

    build_artifacts "${bargs[@]}" || return

    local pargs=(-n "$pname" -D "$datadir" -t "$tag" -u "$guser" -p "$gtoken")

    publish_github "${pargs[@]}" || return

    local dargs=(-n "$pname" -s "$sdesc" -d "$pdir" -c "$cfgdir" -t "$tag" -U "$duser" -P "$dtoken")

    publish_docker_image "${dargs[@]}" || return

    info "Complete."

    return 0
}

main "$@"

