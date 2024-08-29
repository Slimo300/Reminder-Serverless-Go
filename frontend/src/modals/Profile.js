import React, { useState, useRef } from "react";
import { Modal, ModalHeader, ModalBody } from 'reactstrap';

import axiosObject from "../requests/API";

export const ModalUserProfile = ({ toggle, show, phoneNumber, setPhoneNumber }) => {

    const [message, setMessage] = useState("");
    const [showVerification, setShowVerification] = useState(false);

    const [newPhoneNumber, setNewPhoneNumber] = useState("");
    const [verificationCode, setVerificationCode] = useState("");
    const newPhoneNumberInput = useRef(null);

    const updateNumber = async (e) => {
        e.preventDefault();

        try {
            let response = await axiosObject.post("/update-phone-number", {
                phone_number: newPhoneNumber,
            })

            setShowVerification(true);
            setMessage(response.data.message);
        } catch(err) {
            setMessage(err.message)
        } finally {
            setTimeout(() => {
                setMessage("");
            }, 3000);
        }
    };

    const verifyNumber = async (e) => {
        e.preventDefault();

        try {
            let response = await axiosObject.post("/verify-phone-number", {
                verification_code: verificationCode,
            })

            setShowVerification(false);
            setPhoneNumber(newPhoneNumber)
            setMessage(response.data.message);
        } catch(err) {
            setMessage(err.message)
        } finally {
            setTimeout(() => {
                setMessage("");
            }, 3000);
        }
    }

    return (
        <Modal id="buy" tabIndex="-1" role="dialog" isOpen={show} toggle={toggle}>
            <div role="document">
                <ModalHeader toggle={toggle} className="bg-dark text-primary text-center">
                    User Profile
                </ModalHeader>
                <ModalBody>
                    <div className="container">
                        <div className="row d-flex justify-content-center">
                            <div className="text-center card-box">
                                <div className="member-card">
                                    {message}
                                    {
                                        showVerification
                                        ?
                                        <form className="mt-4" onSubmit={verifyNumber}>
                                            <div className="mb-3 text-center">
                                                <label className="form-label">Verification Code</label>
                                                <input type="text" className="form-control" onChange={(e) => setVerificationCode(e.target.value)}/>
                                            </div>
                                            <div className="form-row text-center">
                                                <div className="col-12 mt-2">
                                                    <button type="submit" className="btn btn-dark btn-large text-primary">Verify Number</button>
                                                </div>
                                            </div>
                                        </form>
                                        :
                                        <form className="mt-4" onSubmit={updateNumber}>
                                            <div className="mb-3 text-center">
                                                <label className="form-label">Phone Number</label>
                                                <input type="text" className="form-control" value={phoneNumber} disabled />
                                            </div>
                                            <div className="mb-3 text-center">
                                                <label className="form-label">Change Phone Number</label>
                                                <input type="text" className="form-control" ref={newPhoneNumberInput} onChange={(e) => setNewPhoneNumber(e.target.value)} />
                                            </div>
                                            <div className="form-row text-center">
                                                <div className="col-12 mt-2">
                                                    <button type="submit" className="btn btn-dark btn-large text-primary">Update Number</button>
                                                </div>
                                            </div>
                                        </form>
                                    }
                                </div>
                            </div>
                        </div>
                    </div>
                </ModalBody>
            </div>
        </Modal>
    );
} 