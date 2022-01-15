export const hasAdminSecret = () => {
    return (<any>window)["__authorizer__"].isOnboardingCompleted === true  
}