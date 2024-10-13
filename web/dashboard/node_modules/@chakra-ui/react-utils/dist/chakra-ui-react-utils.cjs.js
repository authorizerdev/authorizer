'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-react-utils.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-react-utils.cjs.dev.js");
}
