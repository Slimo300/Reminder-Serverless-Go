import React, { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { Modal, ModalHeader, ModalBody } from 'reactstrap';

import { forgotPassword, resetPassword } from '../requests/Cognito';

export const ModalForgotPassword = ({ toggle, show }) => {

    const [email, setEmail] = useState("");
    const [resetCode, setResetCode] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [repeatNewPassword, setRepeatNewPassword] = useState("");

    const [showReset, setShowReset] = useState(false);
    const [redirect, setRedirect] = useState(false);
    const [msg, setMsg] = useState("");


    const submitForgotPassword = async(e) => {
        e.preventDefault();

        forgotPassword(email, (err) => {
            if (err) {
                setMsg(err);
            }

            setShowReset(true);
        })
    };

    const submitResetPassword = async(e) => {
        e.preventDefault();

        if (newPassword !== repeatNewPassword) {
            setMsg("Passwords don't match");
            return;
        }

        resetPassword(email, resetCode, newPassword, (err) => {
            if (err) {
                setMsg(err);
                return;
            }

            setRedirect(true);
        })
    };


    if (redirect) return <Navigate to="/login?message=Password+changed"/>;

    return (
        <Modal id="buy" tabIndex="-1" role="dialog" isOpen={ show } toggle={ toggle }>
            <div role="document">
                <ModalHeader toggle={ toggle } className="bg-dark text-primary text-center">
                    Forgot Password?
                </ModalHeader>
                <ModalBody>
                    <div>
                        {msg!==""?<h5 className="mb-4 text-danger">{ msg }</h5>:null}
                        {!showReset?
                        <form onSubmit={ submitForgotPassword }>
                            <div className="form-group">
                                <label>Type in your email:</label>
                                <input type="text" className="form-control" onChange={(e)=>{ setEmail(e.target.value) }}/>
                            </div>
                            <div className="form-row text-center">
                                <div className="col-12 mt-2">
                                    <button type="submit" className="btn btn-dark btn-large text-primary">Send</button>
                                </div>
                            </div>
                        </form>
                        :
                        <form onSubmit={ submitResetPassword }>
                            <div className="form-group">
                                <label>Reset Code:</label>
                                <input type="text" className="form-control" onChange={(e)=>{ setResetCode(e.target.value) }}/>
                            </div>
                            <div className="form-group">
                                <label>New Password:</label>
                                <input type="password" className="form-control" onChange={(e)=>{ setNewPassword(e.target.value) }}/>
                            </div>
                            <div className="form-group">
                                <label>Repeat New Password:</label>
                                <input type="password" className="form-control" onChange={(e)=>{ setRepeatNewPassword(e.target.value) }}/>
                            </div>
                            <div className="form-row text-center">
                                <div className="col-12 mt-2">
                                    <button type="submit" className="btn btn-dark btn-large text-primary">Send</button>
                                </div>
                            </div>
                        </form>}
                    </div>
                </ModalBody>
            </div>
        </Modal>
    );
} 