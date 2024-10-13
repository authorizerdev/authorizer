'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-descendant.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-descendant.cjs.dev.js");
}
