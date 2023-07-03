#!/bin/bash
function checkExecAsRoot() {
    if [ "$EUID" -ne 0 ]; then
        echo "please run as root"
        exit 1
    fi
}

function checkInstalledJq() {
    dpkg -l jq &> /dev/null
    if [ $? -eq 0 ]; then
    echo "✔ jq command is installed"
    else
        echo "jq command is no installed..."
        apt install -y jq
    fi
}

function checkExistCorrectFwConfig() {
    pathToConfig=$1
    conf="fwconfig.json"
    if [ ! -e $pathToConfig ]; then
        echo "config file ($pathToConfig) does not exist!"
        exit 1
    fi

    trust_if=`cat $conf | jq -r ".interfaces.trust_zone"`
    untrust_if=`cat $conf | jq -r ".interfaces.untrust_zone"`

    shouldExit=0
    if [[ "$trust_if" == "null" ]]; then
        echo "the key 'interfaces.trust_zone' is null!"
        shouldExit=1
    fi

    if [[ "$untrust_if" == "null" ]]; then
        echo "the key 'interfaces.untrust_zone' is null!"
        shouldExit=1
    fi

    if [ "$shouldExit" -ne 0 ]; then
        exit 1
    fi
}

function createNftConf() {
    conf="fwconfig.json"
    tmpl="fw-template.rule"
    trust_if_nums=`cat fwconfig.json | jq ".interfaces.trust_zone | length"`
    cat fwconfig.json | jq -r ".interfaces.trust_zone[]" > _trusts.tmp
    untrust_if=`cat fwconfig.json | jq -r ".interfaces.untrust_zone"`
    mgmtaddr_nums=`cat fwconfig.json | jq ".mgmtaddr | length"`
    cat fwconfig.json | jq -r ".mgmtaddr[]" > _addrs.tmp
    fwd=""
    allow_addr=""
    tzone=""
    uzone=""

    PRE_IFS=$IFS
    IFS=$'\n';
    for line in `cat _trusts.tmp`
    do
        fwd+="oifname $line jump ZONE_TRUST;\n\t\t"
        tzone+="iifname $line return;\n\t\t"
        uzone+="iifname $line jump PAIR_trust_to_untrust;\n\t\t"
    done
    for line in `cat _addrs.tmp`
    do
        line=$(echo "$line" | sed 's/\//\\\//g')
        allow_addr+="ip6 saddr $line accept;\n\t\t"
    done
    
    cat $tmpl \
    | sed -e s/\#Allowed_Address_PLACE/"$allow_addr"/g \
    | sed -e s/\#FWD_TRUST_IF_PLACE/"$fwd"/g \
    | sed -e s/\#ZONE_TR_TRUST_IF_PLACE/"$tzone"/g \
    | sed -e s/\#ZONE_UTR_TRUST_IF_PLACE/"$uzone"/g \
    | sed -e s/\{UNTRUST_IF_NAME\}/"$untrust_if"/g > fw.rule
    IFS=$PRE_IFS
    rm _trusts.tmp
    rm _addrs.tmp
}  


function reloadFwRule() {
    netns=`cat $conf | jq -r ".netns"`
    ip netns exec $netns nft -f ./fw.rule
}

function main() {
    checkExecAsRoot
    checkInstalledJq
    checkExistCorrectFwConfig
    createNftConf
    # reloadFwRule
}

main