import { CognitoUserPool, CognitoUser, CognitoUserAttribute, AuthenticationDetails  } from "amazon-cognito-identity-js";

const userPool = new CognitoUserPool({
    UserPoolId: process.env.COGNITO_USER_POOL_ID,
    ClientId: process.env.COGNITO_CLIENT_ID,
});

export const createUser = (username, phone_number, password, callback) => {
    const attributeList = [
        new CognitoUserAttribute({
            Name: "phone_number",
            Value: phone_number
        })
    ];

    userPool.signUp(username, password, attributeList, null, callback);
};

export const verifyUser = (email, verifyCode, callback) => {
    const userData = {
        Username: email,
        Pool: userPool
    };
    const cognitoUser = new CognitoUser(userData);
    cognitoUser.confirmRegistration(verifyCode, true, callback);
}

export const authenticateUser = (email, password, callback) => {
    const authData = {
        Username: email,
        Password: password,
    };

    const authDetails = new AuthenticationDetails(authData);
    const userData = {
        Username: email,
        Pool: userPool,
    };

    const cognitoUser = new CognitoUser(userData);
    cognitoUser.authenticateUser(authDetails, {
        onSuccess: result => {
            callback(null, result);
        },
        onFailure: err => {
            callback(err);
        },
    });
};

export const signOut = () => {
    userPool.getCurrentUser().signOut();
};

export const getCurrentUser = callback => {
    const cognitoUser = userPool.getCurrentUser();
    if (!cognitoUser) return false;

    cognitoUser.getSession((err) => {
        if (err) {
            callback(err)
            return;
        }

        cognitoUser.getUserAttributes((err, attributes) => {
            if (err) {
                callback(err);
                return;
            }
            callback(null, attributes);
        })
    });
};

export const forgotPassword = (email, callback) => {
    const userData = {
        Username: email,
        Pool: userPool,
    };

    const cognitoUser = new CognitoUser(userData);

    cognitoUser.forgotPassword({
        onSuccess: () => {
            callback(null);
        },
        onFailure: err => {
            callback(err);
        },
        inputVerificationCode: () => {
            callback(null);
        }
    })
}

export const resetPassword = (email, resetCode, newPassword, callback) => {
    const userData = {
        Username: email,
        Pool: userPool
    };

    const cognitoUser = new CognitoUser(userData);

    cognitoUser.confirmPassword(resetCode, newPassword, {
        onSuccess: () => {
            callback(null);
        },
        onFailure: err => {
            callback(err);
        }
    });

}

export const getIDToken = () => {
    const cognitoUser = userPool.getCurrentUser();

    let token;
    cognitoUser.getSession((err, session) => {
        if (err) {
            console.log(err);
            return;
        }
        token = session.getIdToken().getJwtToken();
    });

    return token;
};
