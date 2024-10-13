'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var themeTools = require('@chakra-ui/theme-tools');

/**
 * **Accordion anatomy**
 * - Item: the accordion item contains the button and panel
 * - Button: the button is the trigger for the panel
 * - Panel: the panel is the content of the accordion item
 * - Icon: the expanded/collapsed icon
 */

var accordionAnatomy = themeTools.anatomy("accordion").parts("container", "item", "button", "panel").extend("icon");
/**
 * **Alert anatomy**
 * - Title: the alert's title
 * - Description: the alert's description
 * - Icon: the alert's icon
 */

var alertAnatomy = themeTools.anatomy("alert").parts("title", "description", "container").extend("icon");
/**
 * **Avatar anatomy**
 * - Container: the container for the avatar
 * - Label: the avatar initials text
 * - Excess Label: the label or text that represents excess avatar count.
 * Typically used in avatar groups.
 * - Group: the container for the avatar group
 */

var avatarAnatomy = themeTools.anatomy("avatar").parts("label", "badge", "container").extend("excessLabel", "group");
/**
 * **Breadcrumb anatomy**
 * - Item: the container for a breadcrumb item
 * - Link: the element that represents the breadcrumb link
 * - Container: the container for the breadcrumb items
 * - Separator: the separator between breadcrumb items
 */

var breadcrumbAnatomy = themeTools.anatomy("breadcrumb").parts("link", "item", "container").extend("separator");
var buttonAnatomy = themeTools.anatomy("button").parts();
var checkboxAnatomy = themeTools.anatomy("checkbox").parts("control", "icon", "container").extend("label");
var circularProgressAnatomy = themeTools.anatomy("progress").parts("track", "filledTrack").extend("label");
var drawerAnatomy = themeTools.anatomy("drawer").parts("overlay", "dialogContainer", "dialog").extend("header", "closeButton", "body", "footer");
var editableAnatomy = themeTools.anatomy("editable").parts("preview", "input");
var formAnatomy = themeTools.anatomy("form").parts("container", "requiredIndicator", "helperText");
var formErrorAnatomy = themeTools.anatomy("formError").parts("text", "icon");
var inputAnatomy = themeTools.anatomy("input").parts("addon", "field", "element");
var listAnatomy = themeTools.anatomy("list").parts("container", "item", "icon");
var menuAnatomy = themeTools.anatomy("menu").parts("button", "list", "item").extend("groupTitle", "command", "divider");
var modalAnatomy = themeTools.anatomy("modal").parts("overlay", "dialogContainer", "dialog").extend("header", "closeButton", "body", "footer");
var numberInputAnatomy = themeTools.anatomy("numberinput").parts("root", "field", "stepperGroup", "stepper");
var pinInputAnatomy = themeTools.anatomy("pininput").parts("field");
var popoverAnatomy = themeTools.anatomy("popover").parts("content", "header", "body", "footer").extend("popper", "arrow", "closeButton");
var progressAnatomy = themeTools.anatomy("progress").parts("label", "filledTrack", "track");
var radioAnatomy = themeTools.anatomy("radio").parts("container", "control", "label");
var selectAnatomy = themeTools.anatomy("select").parts("field", "icon");
var sliderAnatomy = themeTools.anatomy("slider").parts("container", "track", "thumb", "filledTrack");
var statAnatomy = themeTools.anatomy("stat").parts("container", "label", "helpText", "number", "icon");
var switchAnatomy = themeTools.anatomy("switch").parts("container", "track", "thumb");
var tableAnatomy = themeTools.anatomy("table").parts("table", "thead", "tbody", "tr", "th", "td", "tfoot", "caption");
var tabsAnatomy = themeTools.anatomy("tabs").parts("root", "tab", "tablist", "tabpanel", "tabpanels", "indicator");
/**
 * **Tag anatomy**
 * - Container: the container for the tag
 * - Label: the text content of the tag
 * - closeButton: the close button for the tag
 */

var tagAnatomy = themeTools.anatomy("tag").parts("container", "label", "closeButton");

exports.accordionAnatomy = accordionAnatomy;
exports.alertAnatomy = alertAnatomy;
exports.avatarAnatomy = avatarAnatomy;
exports.breadcrumbAnatomy = breadcrumbAnatomy;
exports.buttonAnatomy = buttonAnatomy;
exports.checkboxAnatomy = checkboxAnatomy;
exports.circularProgressAnatomy = circularProgressAnatomy;
exports.drawerAnatomy = drawerAnatomy;
exports.editableAnatomy = editableAnatomy;
exports.formAnatomy = formAnatomy;
exports.formErrorAnatomy = formErrorAnatomy;
exports.inputAnatomy = inputAnatomy;
exports.listAnatomy = listAnatomy;
exports.menuAnatomy = menuAnatomy;
exports.modalAnatomy = modalAnatomy;
exports.numberInputAnatomy = numberInputAnatomy;
exports.pinInputAnatomy = pinInputAnatomy;
exports.popoverAnatomy = popoverAnatomy;
exports.progressAnatomy = progressAnatomy;
exports.radioAnatomy = radioAnatomy;
exports.selectAnatomy = selectAnatomy;
exports.sliderAnatomy = sliderAnatomy;
exports.statAnatomy = statAnatomy;
exports.switchAnatomy = switchAnatomy;
exports.tableAnatomy = tableAnatomy;
exports.tabsAnatomy = tabsAnatomy;
exports.tagAnatomy = tagAnatomy;
