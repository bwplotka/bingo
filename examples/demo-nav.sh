#!/usr/bin/env bash

# Source & docs: https://github.com/bwplotka/demo-nav

# Script options (define those variables before registering commands):
#
# The speed to "type" the text (Default: 0 so no typing view).
# TYPE_SPEED=40
#
# If false next command will be shown only after enter (Default: false).
# IMMEDIATE_REVEAL=true
#
# If true prefix line with number; easier to navigate (Default: false).
# NUMS=true
#
# If NUMS = false this prefix will be used (Default: '').
# PREFIX="CustomPrefix"
#
# Color vars for pretty prompts.
# Feel free to use those colors in registered commands.
BLACK="\033[0;30m"
YELLOW="\033[1;33m"
BLUE="\033[0;34m"
GREEN="\033[0;32m"
CYAN="\033[0;36m"
RED="\033[0;31m"
PURPLE="\033[0;35m"
BROWN="\033[0;33m"
WHITE="\033[1;37m"
COLOR_RESET="\033[0m"

# Shortcuts bindings.
NEXT_KEY=$'\x6E' # n
PREV_KEY=$'\x70' # p
BEGIN_KEY=$'\x62' # b
END_KEY=$'\x65' # e
QUIT_KEY=$'\x71' # q
INVOKE_KEY=$'\x0' # enter

# Variables.
PRINT=()
CMDS=()
CLEAN_AFTER=()

# Strip ANSI escape codes/sequences [$1: input string, $2: target variable]
function strip_escape_codes_and_comments() {
    local _input="$1" _i _j _token _escape=0
    local -n _output="$2"; _output=""
    for (( _i=0; _i < ${#_input}; _i++ )); do
       if (( ${_escape} == 1 )); then
            if [[ "${_input:_i:1}" =~ [a-zA-Z] ]]; then
                _escape=0
            fi
            continue
       fi
       if [[ "${_input:_i:5}" == "\033[" ]]; then
            _escape=1
            continue
        fi

        if [[ "${_input:_i:1}" == '#' ]]; then
            break
        fi
        _output+="${_input:_i:1}"
    done
}

##
# Registers a command into navigable script. Order of registration matters.
#
# Takes 1 or 2 parameters:
# 1) The string command to show.
# 2) Optionally: The string command to run. If empty, parameter 1 is used.
#
# usage:
#
#   r "ls -l"
#   r "list me please" "ls -l"
##
function r() {
  PRINT+=("${1}")

  TO_RUN="${2:-${1}}"

  # Sanitize.
  strip_escape_codes_and_comments "${TO_RUN}" TO_RUN_SANITIZED
  CMDS+=("${TO_RUN_SANITIZED}")
  CLEAN_AFTER+=(false)
}

##
# Same as 'r' but removes the command *AFTER* the execution.
##
function rc() {
  r "$1" "$2"

  CLEAN_AFTER[-1]=true
}

##
# Runs in a mode that enables easy navigation of the
# commands in the sequential manner.
#
# TODO(bwplotka): Add search (ctlr+r) functionality
##
function navigate() {
  CONTINUE=${1-false}

  curr=0
  if ${CONTINUE} && [[ -f ./.demo-last-step ]]; then
    curr=$(< ./.demo-last-step)
  fi

  while true
  do
    # Check boundaries.
    if (( ${curr} < 0 )); then
      curr=0
    fi
    if (( ${curr} >= ${#CMDS[@]} )); then
      let curr="${#CMDS[@]} - 1"
    fi

    print=${PRINT[${curr}]}
    if [[ ${print} == "" ]]; then
        print=${CMDS[${curr}]}
    fi

    prefix="${PREFIX}"
    if ${NUMS}; then
        prefix="${curr}) "
    fi
    # Make sure input will not break the print.
    stty -echo
    if [[ -z $TYPE_SPEED ]]; then
      echo -en "${prefix}${YELLOW}$print${COLOR_RESET}"
    else
      echo -en "${prefix}${YELLOW}$print${COLOR_RESET}" | pv -qL $[$TYPE_SPEED+(-2 + RANDOM%5)];
    fi
    stty echo

    # Ignore accidentally buffered input (introduces 0.5 input lag).
    read -rst 0.3 -n 10000 discard

    # Allow for interactive navigation.
    read -rsn1 input
    case "${input}" in
    ${BEGIN_KEY})
      # Skip this command and move to beginning.
      curr=0
      echo -en "\033[2K\r"
      ;;
    ${END_KEY})
      # Skip this command and move to the end.
      let curr="${#CMDS[@]} - 1"
      echo -en "\033[2K\r"
      ;;
    ${NEXT_KEY})
      # Skip this command and move to next.
      ((curr++))
      echo -en "\033[2K\r"
      ;;
    ${PREV_KEY})
      # Skip this command and move to previous.
      ((curr--))
      echo -en "\033[2K\r"
      ;;
    ${INVOKE_KEY})
      # enter - Eval this and move to next.
      if ${CLEAN_AFTER[${curr}]}; then
        echo -en "\033[2K\r"
      else
        echo ""
      fi
      eval "${CMDS[${curr}]}"
      ((curr++))

      if ${IMMEDIATE_REVEAL}; then
         # Wait for enter at the end.
      read -rst 0.3 -n 10000 discard
      read -rsn1 input
        case ${input} in
        ${NEXT_KEY})
          ((curr++))
          echo -en "\033[2K\r"
          ;;
        ${PREV_KEY})
          ((curr--))
          echo -en "\033[2K\r"
          ;;
        ${QUIT_KEY})
          echo ""
          echo "Bye!"
          exit 0
          ;;
        esac
      fi
      ;;
    ${QUIT_KEY})
     # q - Quit.
      echo ""
      echo "Bye!"
      exit 0
      ;;
    *)
    # Print again, not supported input.
      echo -en "\r"
      ;;
    esac
    echo ${curr} > ./.demo-last-step
  done
}

# Internal function for checking pv tool that is used to simulate typing.
function _check_pv() {
  command -v pv >/dev/null 2>&1 || {
    echo ""
    echo "'pv' tool is required, but it's not installed. Aborting." >&2;
    echo ""
    echo -e "${COLOR_RESET}Installing pv:"
    echo ""
    echo -e "${BLUE}Mac:${COLOR_RESET} $ brew install pv"
    echo ""
    echo -e "${BLUE}Other:${COLOR_RESET} http://www.ivarch.com/programs/pv.shtml"
    echo -e "${COLOR_RESET}"
    exit 1;
  }
}

if ! [[ -z ${TYPE_SPEED} ]]; then
    _check_pv
fi