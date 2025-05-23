"use strict";

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; var ownKeys = Object.keys(source); if (typeof Object.getOwnPropertySymbols === 'function') { ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) { return Object.getOwnPropertyDescriptor(source, sym).enumerable; })); } ownKeys.forEach(function (key) { _defineProperty(target, key, source[key]); }); } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }

/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 *
 * @format
 * 
 * @emails oncall+draft_js
 */
var DraftEntityInstance = require("./DraftEntityInstance");

var Immutable = require("immutable");

var invariant = require("fbjs/lib/invariant");

var uuid = require("./uuid");

var Map = Immutable.Map;
var instances = Map();
var instanceKey = uuid();
/**
 * Temporary utility for generating the warnings
 */

function logWarning(oldMethodCall, newMethodCall) {
  console.warn('WARNING: ' + oldMethodCall + ' will be deprecated soon!\nPlease use "' + newMethodCall + '" instead.');
}

/**
 * A "document entity" is an object containing metadata associated with a
 * piece of text in a ContentBlock.
 *
 * For example, a `link` entity might include a `uri` property. When a
 * ContentBlock is rendered in the browser, text that refers to that link
 * entity may be rendered as an anchor, with the `uri` as the href value.
 *
 * In a ContentBlock, every position in the text may correspond to zero
 * or one entities. This correspondence is tracked using a key string,
 * generated via DraftEntity.create() and used to obtain entity metadata
 * via DraftEntity.get().
 */
var DraftEntity = {
  /**
   * WARNING: This method will be deprecated soon!
   * Please use 'contentState.getLastCreatedEntityKey' instead.
   * ---
   * Get the random key string from whatever entity was last created.
   * We need this to support the new API, as part of transitioning to put Entity
   * storage in contentState.
   */
  getLastCreatedEntityKey: function getLastCreatedEntityKey() {
    logWarning('DraftEntity.getLastCreatedEntityKey', 'contentState.getLastCreatedEntityKey');
    return DraftEntity.__getLastCreatedEntityKey();
  },

  /**
   * WARNING: This method will be deprecated soon!
   * Please use 'contentState.createEntity' instead.
   * ---
   * Create a DraftEntityInstance and store it for later retrieval.
   *
   * A random key string will be generated and returned. This key may
   * be used to track the entity's usage in a ContentBlock, and for
   * retrieving data about the entity at render time.
   */
  create: function create(type, mutability, data) {
    logWarning('DraftEntity.create', 'contentState.createEntity');
    return DraftEntity.__create(type, mutability, data);
  },

  /**
   * WARNING: This method will be deprecated soon!
   * Please use 'contentState.addEntity' instead.
   * ---
   * Add an existing DraftEntityInstance to the DraftEntity map. This is
   * useful when restoring instances from the server.
   */
  add: function add(instance) {
    logWarning('DraftEntity.add', 'contentState.addEntity');
    return DraftEntity.__add(instance);
  },

  /**
   * WARNING: This method will be deprecated soon!
   * Please use 'contentState.getEntity' instead.
   * ---
   * Retrieve the entity corresponding to the supplied key string.
   */
  get: function get(key) {
    logWarning('DraftEntity.get', 'contentState.getEntity');
    return DraftEntity.__get(key);
  },

  /**
   * Get all the entities in the content state.
   */
  __getAll: function __getAll() {
    return instances;
  },

  /**
   * Load the entity map with the given set of entities.
   */
  __loadWithEntities: function __loadWithEntities(entities) {
    instances = entities;
    instanceKey = uuid();
  },

  /**
   * WARNING: This method will be deprecated soon!
   * Please use 'contentState.mergeEntityData' instead.
   * ---
   * Entity instances are immutable. If you need to update the data for an
   * instance, this method will merge your data updates and return a new
   * instance.
   */
  mergeData: function mergeData(key, toMerge) {
    logWarning('DraftEntity.mergeData', 'contentState.mergeEntityData');
    return DraftEntity.__mergeData(key, toMerge);
  },

  /**
   * WARNING: This method will be deprecated soon!
   * Please use 'contentState.replaceEntityData' instead.
   * ---
   * Completely replace the data for a given instance.
   */
  replaceData: function replaceData(key, newData) {
    logWarning('DraftEntity.replaceData', 'contentState.replaceEntityData');
    return DraftEntity.__replaceData(key, newData);
  },
  // ***********************************WARNING******************************
  // --- the above public API will be deprecated in the next version of Draft!
  // The methods below this line are private - don't call them directly.

  /**
   * Get the random key string from whatever entity was last created.
   * We need this to support the new API, as part of transitioning to put Entity
   * storage in contentState.
   */
  __getLastCreatedEntityKey: function __getLastCreatedEntityKey() {
    return instanceKey;
  },

  /**
   * Create a DraftEntityInstance and store it for later retrieval.
   *
   * A random key string will be generated and returned. This key may
   * be used to track the entity's usage in a ContentBlock, and for
   * retrieving data about the entity at render time.
   */
  __create: function __create(type, mutability, data) {
    return DraftEntity.__add(new DraftEntityInstance({
      type: type,
      mutability: mutability,
      data: data || {}
    }));
  },

  /**
   * Add an existing DraftEntityInstance to the DraftEntity map. This is
   * useful when restoring instances from the server.
   */
  __add: function __add(instance) {
    instanceKey = uuid();
    instances = instances.set(instanceKey, instance);
    return instanceKey;
  },

  /**
   * Retrieve the entity corresponding to the supplied key string.
   */
  __get: function __get(key) {
    var instance = instances.get(key);
    !!!instance ? process.env.NODE_ENV !== "production" ? invariant(false, 'Unknown DraftEntity key: %s.', key) : invariant(false) : void 0;
    return instance;
  },

  /**
   * Entity instances are immutable. If you need to update the data for an
   * instance, this method will merge your data updates and return a new
   * instance.
   */
  __mergeData: function __mergeData(key, toMerge) {
    var instance = DraftEntity.__get(key);

    var newData = _objectSpread({}, instance.getData(), toMerge);

    var newInstance = instance.set('data', newData);
    instances = instances.set(key, newInstance);
    return newInstance;
  },

  /**
   * Completely replace the data for a given instance.
   */
  __replaceData: function __replaceData(key, newData) {
    var instance = DraftEntity.__get(key);

    var newInstance = instance.set('data', newData);
    instances = instances.set(key, newInstance);
    return newInstance;
  }
};
module.exports = DraftEntity;