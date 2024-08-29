import React, { useState } from "react";
import { Navigate } from "react-router-dom";

import { ModalForgotPassword } from "../modals/ForgotPassword";

import useQuery from "../hooks/useQuery";
import { authenticateUser } from "../requests/Cognito";

const SignInForm = ({ loggedIn, setLoggedIn }) => {

  const [forgotPasswordShow, setForgotPasswordShow] = useState(false);
  const toggleForgotPassword = () => {
      setForgotPasswordShow(!forgotPasswordShow);
  };

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const [message, setMessage] = useState('');
  const msg = useQuery().get("message");

  const submitLogin = async (e) => {
    e.preventDefault();

    if (email.trim() === "") {
        setMessage("Email cannot be blank");
        return;
    }
    if (password.trim() === "") {
        setMessage("Password cannot be blank");
        return;
    }

    authenticateUser(email, password, (err, result) => {
      if (err) {
        setMessage(err);
        return;
      }
      setLoggedIn(true);
      window.localStorage.setItem("LOGGED_IN", true);
    });
    
  }

  if (loggedIn) return <Navigate to="/"/>;

  return (
    <div className="container pt-4 mt-4">
      <div className="mt-5 d-flex justify-content-center">
        <div className="mt-5 row">
          <form onSubmit={submitLogin}>
            <div className="display-3 mb-4 text-center text-primary"> Log In</div>
            <div className="mb-3 text-center text-danger">{message!==""?message:msg}</div>
            <div className="mb-3 text-center">
              <label className="form-label">Username</label>
              <input type="text" className="form-control" onChange={e => setEmail(e.target.value)}/>
            </div>
            <div className="mb-3 text-center">
              <label className="form-label">Password</label>
              <input type="password" className="form-control" onChange={e => setPassword(e.target.value)}/>
            </div>
            <div className="text-center">
              <button type="submit" className="btn btn-primary text-center">Submit</button>
            </div>
            <div className="display-5 mt-4 text-center text-primary"><a href="/register">or Register</a></div>
            <div className="display-5 mt-4 text-center text-primary"><a href="#" onClick={toggleForgotPassword}>Forgot your password?</a></div>
          </form>
        </div>
      </div>
      <ModalForgotPassword show={forgotPasswordShow} toggle={toggleForgotPassword} />
    </div>
   )
}

export default SignInForm;