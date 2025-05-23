import React from 'react';
import PropTypes from 'prop-types';
import withSideEffect from 'react-clientside-effect';
import { moveFocusInside, focusInside, focusIsHidden, expandFocusableNodes, getFocusableNodes, focusNextElement, focusPrevElement, focusFirstElement, focusLastElement, captureFocusRestore } from 'focus-lock';
import { deferAction, extractRef } from './util';
import { mediumFocus, mediumBlur, mediumEffect } from './medium';
var focusOnBody = function focusOnBody() {
  return document && document.activeElement === document.body;
};
var isFreeFocus = function isFreeFocus() {
  return focusOnBody() || focusIsHidden();
};
var lastActiveTrap = null;
var lastActiveFocus = null;
var tryRestoreFocus = function tryRestoreFocus() {
  return null;
};
var lastPortaledElement = null;
var focusWasOutsideWindow = false;
var windowFocused = false;
var defaultWhitelist = function defaultWhitelist() {
  return true;
};
var focusWhitelisted = function focusWhitelisted(activeElement) {
  return (lastActiveTrap.whiteList || defaultWhitelist)(activeElement);
};
var recordPortal = function recordPortal(observerNode, portaledElement) {
  lastPortaledElement = {
    observerNode: observerNode,
    portaledElement: portaledElement
  };
};
var focusIsPortaledPair = function focusIsPortaledPair(element) {
  return lastPortaledElement && lastPortaledElement.portaledElement === element;
};
function autoGuard(startIndex, end, step, allNodes) {
  var lastGuard = null;
  var i = startIndex;
  do {
    var item = allNodes[i];
    if (item.guard) {
      if (item.node.dataset.focusAutoGuard) {
        lastGuard = item;
      }
    } else if (item.lockItem) {
      if (i !== startIndex) {
        return;
      }
      lastGuard = null;
    } else {
      break;
    }
  } while ((i += step) !== end);
  if (lastGuard) {
    lastGuard.node.tabIndex = 0;
  }
}
var focusWasOutside = function focusWasOutside(crossFrameOption) {
  if (crossFrameOption) {
    return Boolean(focusWasOutsideWindow);
  }
  return focusWasOutsideWindow === 'meanwhile';
};
var checkInHost = function checkInHost(check, el, boundary) {
  return el && (el.host === check && (!el.activeElement || boundary.contains(el.activeElement)) || el.parentNode && checkInHost(check, el.parentNode, boundary));
};
var withinHost = function withinHost(activeElement, workingArea) {
  return workingArea.some(function (area) {
    return checkInHost(activeElement, area, area);
  });
};
var getNodeFocusables = function getNodeFocusables(nodes) {
  return getFocusableNodes(nodes, new Map());
};
var isNotFocusable = function isNotFocusable(node) {
  return !getNodeFocusables([node.parentNode]).some(function (el) {
    return el.node === node;
  });
};
var activateTrap = function activateTrap() {
  var result = false;
  if (lastActiveTrap) {
    var _lastActiveTrap = lastActiveTrap,
      observed = _lastActiveTrap.observed,
      persistentFocus = _lastActiveTrap.persistentFocus,
      autoFocus = _lastActiveTrap.autoFocus,
      shards = _lastActiveTrap.shards,
      crossFrame = _lastActiveTrap.crossFrame,
      focusOptions = _lastActiveTrap.focusOptions,
      noFocusGuards = _lastActiveTrap.noFocusGuards;
    var workingNode = observed || lastPortaledElement && lastPortaledElement.portaledElement;
    if (focusOnBody() && lastActiveFocus && lastActiveFocus !== document.body) {
      if (!document.body.contains(lastActiveFocus) || isNotFocusable(lastActiveFocus)) {
        var newTarget = tryRestoreFocus();
        if (newTarget) {
          newTarget.focus();
        }
      }
    }
    var activeElement = document && document.activeElement;
    if (workingNode) {
      var workingArea = [workingNode].concat(shards.map(extractRef).filter(Boolean));
      var shouldForceRestoreFocus = function shouldForceRestoreFocus() {
        if (!focusWasOutside(crossFrame) || !noFocusGuards || !lastActiveFocus || windowFocused) {
          return false;
        }
        var nodes = getNodeFocusables(workingArea);
        var lastIndex = nodes.findIndex(function (_ref) {
          var node = _ref.node;
          return node === lastActiveFocus;
        });
        return lastIndex === 0 || lastIndex === nodes.length - 1;
      };
      if (!activeElement || focusWhitelisted(activeElement)) {
        if (persistentFocus || shouldForceRestoreFocus() || !isFreeFocus() || !lastActiveFocus && autoFocus) {
          if (workingNode && !(focusInside(workingArea) || activeElement && withinHost(activeElement, workingArea) || focusIsPortaledPair(activeElement, workingNode))) {
            if (document && !lastActiveFocus && activeElement && !autoFocus) {
              if (activeElement.blur) {
                activeElement.blur();
              }
              document.body.focus();
            } else {
              result = moveFocusInside(workingArea, lastActiveFocus, {
                focusOptions: focusOptions
              });
              lastPortaledElement = {};
            }
          }
          lastActiveFocus = document && document.activeElement;
          if (lastActiveFocus !== document.body) {
            tryRestoreFocus = captureFocusRestore(lastActiveFocus);
          }
          focusWasOutsideWindow = false;
        }
      }
      if (document && activeElement !== document.activeElement && document.querySelector('[data-focus-auto-guard]')) {
        var newActiveElement = document && document.activeElement;
        var allNodes = expandFocusableNodes(workingArea);
        var focusedIndex = allNodes.map(function (_ref2) {
          var node = _ref2.node;
          return node;
        }).indexOf(newActiveElement);
        if (focusedIndex > -1) {
          allNodes.filter(function (_ref3) {
            var guard = _ref3.guard,
              node = _ref3.node;
            return guard && node.dataset.focusAutoGuard;
          }).forEach(function (_ref4) {
            var node = _ref4.node;
            return node.removeAttribute('tabIndex');
          });
          autoGuard(focusedIndex, allNodes.length, +1, allNodes);
          autoGuard(focusedIndex, -1, -1, allNodes);
        }
      }
    }
  }
  return result;
};
var onTrap = function onTrap(event) {
  if (activateTrap() && event) {
    event.stopPropagation();
    event.preventDefault();
  }
};
var onBlur = function onBlur() {
  return deferAction(activateTrap);
};
var onFocus = function onFocus(event) {
  var source = event.target;
  var currentNode = event.currentTarget;
  if (!currentNode.contains(source)) {
    recordPortal(currentNode, source);
  }
};
var FocusWatcher = function FocusWatcher() {
  return null;
};
var FocusTrap = function FocusTrap(_ref5) {
  var children = _ref5.children;
  return /*#__PURE__*/React.createElement("div", {
    onBlur: onBlur,
    onFocus: onFocus
  }, children);
};
FocusTrap.propTypes = process.env.NODE_ENV !== "production" ? {
  children: PropTypes.node.isRequired
} : {};
var onWindowFocus = function onWindowFocus() {
  windowFocused = true;
};
var onWindowBlur = function onWindowBlur() {
  windowFocused = false;
  focusWasOutsideWindow = 'just';
  deferAction(function () {
    focusWasOutsideWindow = 'meanwhile';
  });
};
var attachHandler = function attachHandler() {
  document.addEventListener('focusin', onTrap);
  document.addEventListener('focusout', onBlur);
  window.addEventListener('focus', onWindowFocus);
  window.addEventListener('blur', onWindowBlur);
};
var detachHandler = function detachHandler() {
  document.removeEventListener('focusin', onTrap);
  document.removeEventListener('focusout', onBlur);
  window.removeEventListener('focus', onWindowFocus);
  window.removeEventListener('blur', onWindowBlur);
};
function reducePropsToState(propsList) {
  return propsList.filter(function (_ref6) {
    var disabled = _ref6.disabled;
    return !disabled;
  });
}
var focusLockAPI = {
  moveFocusInside: moveFocusInside,
  focusInside: focusInside,
  focusNextElement: focusNextElement,
  focusPrevElement: focusPrevElement,
  focusFirstElement: focusFirstElement,
  focusLastElement: focusLastElement,
  captureFocusRestore: captureFocusRestore
};
function handleStateChangeOnClient(traps) {
  var trap = traps.slice(-1)[0];
  if (trap && !lastActiveTrap) {
    attachHandler();
  }
  var lastTrap = lastActiveTrap;
  var sameTrap = lastTrap && trap && trap.id === lastTrap.id;
  lastActiveTrap = trap;
  if (lastTrap && !sameTrap) {
    lastTrap.onDeactivation();
    if (!traps.filter(function (_ref7) {
      var id = _ref7.id;
      return id === lastTrap.id;
    }).length) {
      lastTrap.returnFocus(!trap);
    }
  }
  if (trap) {
    lastActiveFocus = null;
    if (!sameTrap || lastTrap.observed !== trap.observed) {
      trap.onActivation(focusLockAPI);
    }
    activateTrap(true);
    deferAction(activateTrap);
  } else {
    detachHandler();
    lastActiveFocus = null;
  }
}
mediumFocus.assignSyncMedium(onFocus);
mediumBlur.assignMedium(onBlur);
mediumEffect.assignMedium(function (cb) {
  return cb(focusLockAPI);
});
export default withSideEffect(reducePropsToState, handleStateChangeOnClient)(FocusWatcher);