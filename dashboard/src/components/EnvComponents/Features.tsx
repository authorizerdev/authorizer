import React from "react";
import { Divider, Flex, Stack, Text } from "@chakra-ui/react";
import InputField from "../InputField";
import { SwitchInputType } from "../../constants";

const Features = ({ variables, setVariables }: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
        Disable Features
      </Text>
      <Stack spacing={6}>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable secure app cookie:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_APP_COOKIE_SECURE}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable secure admin cookie:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_ADMIN_COOKIE_SECURE}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Login Page:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_LOGIN_PAGE}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Email Verification:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_EMAIL_VERIFICATION}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Magic Login Link:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_MAGIC_LINK_LOGIN}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Basic Authentication:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_BASIC_AUTHENTICATION}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Sign Up:</Text>
          </Flex>
          <Flex justifyContent="start" mb={3}>
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_SIGN_UP}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Strong Password:</Text>
          </Flex>
          <Flex justifyContent="start" mb={3}>
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_STRONG_PASSWORD}
            />
          </Flex>
        </Flex>
        <Flex alignItems="center">
          <Flex w="100%" alignItems="baseline" flexDir="column">
            <Text fontSize="sm">
              Disable Multi Factor Authentication (MFA):
            </Text>
            <Text fontSize="x-small">
              Note: Enabling this will ignore Enforcing MFA shown below and will
              also ignore the user MFA setting.
            </Text>
          </Flex>
          <Flex justifyContent="start" mb={3}>
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_MULTI_FACTOR_AUTHENTICATION}
            />
          </Flex>
        </Flex>
      </Stack>
      <Divider paddingY={5} />
      <Text fontSize="md" paddingTop={5} fontWeight="bold" mb={5}>
        Enable Features
      </Text>
      <Stack spacing={6}>
        <Flex alignItems="center">
          <Flex w="100%" alignItems="baseline" flexDir="column">
            <Text fontSize="sm">
              Enforce Multi Factor Authentication (MFA):
            </Text>
            <Text fontSize="x-small">
              Note: If you disable enforcing after it was enabled, it will still
              keep MFA enabled for older users.
            </Text>
          </Flex>
          <Flex justifyContent="start" mb={3}>
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.ENFORCE_MULTI_FACTOR_AUTHENTICATION}
            />
          </Flex>
        </Flex>
      </Stack>
    </div>
  );
};

export default Features;
