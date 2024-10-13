'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-avatar.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-avatar.cjs.dev.js");
}
