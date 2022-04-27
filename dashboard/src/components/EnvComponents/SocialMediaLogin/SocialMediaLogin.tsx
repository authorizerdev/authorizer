import React from "react";
import { FaGoogle, FaGithub, FaFacebookF } from "react-icons/fa";
import {
  Box,
  Divider,
  Flex,
  Stack,
  Center,
  Text,
  useMediaQuery,
} from "@chakra-ui/react";
import { TextInputType, HiddenInputType } from "../../../constants";
import InputField from "../../InputField";
const SocialMediaLogin = ({
  variables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
}: any) => {
  const [isNotSmallerScreen] = useMediaQuery("(min-width:600px)");
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={4}>
        Social Media Logins
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Center
            w={isNotSmallerScreen ? "55px" : "35px"}
            h="35px"
            marginRight="1.5%"
            border="1px solid #ff3e30"
            borderRadius="5px"
          >
            <FaGoogle style={{ color: "#ff3e30" }} />
          </Center>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
            marginRight="1.5%"
          >
            <InputField
              borderRadius={7}
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.GOOGLE_CLIENT_ID}
              placeholder="Google Client ID"
            />
          </Center>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
            <InputField
              borderRadius={7}
              variables={variables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.GOOGLE_CLIENT_SECRET}
              placeholder="Google Secret"
            />
          </Center>
        </Flex>
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Center
            w={isNotSmallerScreen ? "55px" : "35px"}
            h="35px"
            marginRight="1.5%"
            border="1px solid #171515"
            borderRadius="5px"
          >
            <FaGithub style={{ color: "#171515" }} />
          </Center>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
            marginRight="1.5%"
          >
            <InputField
              borderRadius={7}
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.GITHUB_CLIENT_ID}
              placeholder="Github Client ID"
            />
          </Center>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
            <InputField
              borderRadius={7}
              variables={variables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.GITHUB_CLIENT_SECRET}
              placeholder="Github Secret"
            />
          </Center>
        </Flex>
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Center
            w={isNotSmallerScreen ? "55px" : "35px"}
            h="35px"
            marginRight="1.5%"
            border="1px solid #3b5998"
            borderRadius="5px"
          >
            <FaFacebookF style={{ color: "#3b5998" }} />
          </Center>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
            marginRight="1.5%"
          >
            <InputField
              borderRadius={7}
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.FACEBOOK_CLIENT_ID}
              placeholder="Facebook Client ID"
            />
          </Center>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
            <InputField
              borderRadius={7}
              variables={variables}
              setVariables={setVariables}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              inputType={HiddenInputType.FACEBOOK_CLIENT_SECRET}
              placeholder="Facebook Secret"
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default SocialMediaLogin;
