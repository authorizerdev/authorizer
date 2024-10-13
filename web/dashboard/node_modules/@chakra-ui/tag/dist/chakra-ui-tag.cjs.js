'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-tag.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-tag.cjs.dev.js");
}
